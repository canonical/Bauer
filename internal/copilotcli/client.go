package copilotcli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
	colorBright = "\033[1m"
)

// isTTY checks if stdout is a terminal (supports colors)
func isTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// formatCopilotOutput wraps text in color codes if terminal supports it
func formatCopilotOutput(text string) string {
	if isTTY() {
		return colorCyan + text + colorReset
	}
	return text
}

// formatCopilotDim formats dimmed text for reasoning/debug output
func formatCopilotDim(text string) string {
	if isTTY() {
		return colorDim + text + colorReset
	}
	return text
}

// formatSummaryOutput wraps text in yellow color for summary sessions
func formatSummaryOutput(text string) string {
	if isTTY() {
		return colorYellow + text + colorReset
	}
	return text
}

// Client wraps the GitHub Copilot SDK client
type Client struct {
	client *copilot.Client
	cwd    string
}

// NewClient creates and initializes a new Copilot client
func NewClient(cwd string) (*Client, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Create Copilot SDK client with default options
	sdkClient := copilot.NewClient(&copilot.ClientOptions{
		// CLIPath:  os.Getenv("COPILOT_CLI_PATH"), // Optional override
		Cwd:      cwd,
		LogLevel: "info", // TODO make configurable - set to error in production
	})

	return &Client{
		client: sdkClient,
		cwd:    cwd,
	}, nil
}

// Start starts the Copilot CLI server
func (c *Client) Start() error {
	slog.Info("Starting Copilot client...")
	if err := c.client.Start(); err != nil {
		return fmt.Errorf("failed to start Copilot client: %w", err)
	}

	// Verify connectivity with ping
	_, err := c.client.Ping("health-check")
	if err != nil {
		c.client.Stop()
		return fmt.Errorf("Copilot client ping failed: %w", err)
	}

	slog.Info("Copilot client started successfully")
	return nil
}

// Stop gracefully stops the Copilot CLI server
func (c *Client) Stop() error {
	slog.Info("Stopping Copilot client...")
	errs := c.client.Stop()
	if len(errs) > 0 {
		for _, err := range errs {
			slog.Error("Error during Copilot client shutdown", slog.String("error", err.Error()))
		}
		return fmt.Errorf("encountered %d errors during shutdown", len(errs))
	}
	slog.Info("Copilot client stopped successfully")
	return nil
}

// ExecuteChunk processes a single chunk prompt using a Copilot session and returns the output
func (c *Client) ExecuteChunk(ctx context.Context, chunkPath string, chunkNumber int, model string) (string, error) {
	slog.Info("Creating Copilot session",
		slog.Int("chunk", chunkNumber),
		slog.String("model", model),
	)

	// Create a session with streaming enabled
	session, err := c.client.CreateSession(&copilot.SessionConfig{
		Model:     model,
		Streaming: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session for chunk %d: %w", chunkNumber, err)
	}
	defer func() {
		if err := session.Destroy(); err != nil {
			slog.Error("Failed to destroy session",
				slog.Int("chunk", chunkNumber),
				slog.String("error", err.Error()),
			)
		}
	}()

	// Set up event handler to stream output
	done := make(chan error, 1)
	var fullOutput string

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		// TODO these 2 events should be only for debugging/verbose logging
		case "assistant.message_delta":
			// Stream incremental content as it comes (in color)
			if event.Data.DeltaContent != nil {
				fmt.Print(formatCopilotOutput(*event.Data.DeltaContent))
				fullOutput += *event.Data.DeltaContent
			}

		case "assistant.reasoning_delta":
			// Stream reasoning content (dimmed)
			if event.Data.DeltaContent != nil {
				fmt.Print(formatCopilotDim(*event.Data.DeltaContent))
				fullOutput += *event.Data.DeltaContent
			}

		case "assistant.message":
			// Add to output and print the final message
			if event.Data.Content != nil {
				fullOutput += *event.Data.Content
				fmt.Println(formatCopilotOutput(*event.Data.Content))
				slog.Debug("Assistant response",
					slog.Int("chunk", chunkNumber),
					slog.String("content", *event.Data.Content),
				)
			}

		case "assistant.reasoning":
			// Add to output and print reasoning in dimmed style
			if event.Data.Content != nil {
				fullOutput += *event.Data.Content
				fmt.Println(formatCopilotDim(*event.Data.Content))
				slog.Debug("Assistant reasoning response",
					slog.Int("chunk", chunkNumber),
					slog.String("content", *event.Data.Content),
				)
			}

		case "session.idle":
			// Session completed successfully
			slog.Info("Session completed",
				slog.Int("chunk", chunkNumber),
			)
			done <- nil

		case "session.error":
			// Session encountered an error
			errMsg := fmt.Sprintf("session error for chunk %d", chunkNumber)
			if event.Data.Error != nil {
				errMsg = fmt.Sprintf("%s: %v", errMsg, event.Data.Error)
			}
			slog.Error("Session error",
				slog.Int("chunk", chunkNumber),
				slog.String("error", errMsg),
			)
			done <- fmt.Errorf("%s", errMsg)

		case "assistant.tool_call":
			// Log tool calls for visibility
			if event.Data.ToolName != nil {
				slog.Debug("Tool called",
					slog.Int("chunk", chunkNumber),
					slog.String("tool", *event.Data.ToolName),
				)
			}
		}
	})

	// Send the prompt with the chunk file as attachment
	// Ensure the path is absolute for reliable access
	absChunkPath, err := filepath.Abs(chunkPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve chunk path: %w", err)
	}

	slog.Info("Sending prompt to Copilot",
		slog.Int("chunk", chunkNumber),
		slog.String("file", absChunkPath),
	)

	_, err = session.Send(copilot.MessageOptions{
		Prompt: fmt.Sprintf("Implement the changes described in @%s. Follow all instructions carefully and apply changes in order.", filepath.Base(chunkPath)),
		Attachments: []copilot.Attachment{
			{
				Type:        copilot.File,
				Path:        absChunkPath,
				DisplayName: fmt.Sprintf("chunk-%d.md", chunkNumber),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to send message for chunk %d: %w", chunkNumber, err)
	}

	// Wait for completion with timeout
	select {
	case err := <-done:
		if err != nil {
			return "", err
		}
		fmt.Println() // Add newline after streaming output
		return fullOutput, nil

	case <-time.After(15 * time.Minute):
		return "", fmt.Errorf("chunk %d timed out after 15 minutes", chunkNumber)

	case <-ctx.Done():
		return "", fmt.Errorf("chunk %d cancelled: %w", chunkNumber, ctx.Err())
	}
}

// ChunkOutput represents output from a chunk execution
type ChunkOutput struct {
	ChunkNumber int
	Output      string
	Duration    time.Duration
}

// GenerateSummary creates a summary session with all chunk outputs
func (c *Client) GenerateSummary(ctx context.Context, outputs []ChunkOutput, model string) error {
	slog.Info("Creating summary session", slog.String("model", model))

	// Create a session with streaming enabled
	session, err := c.client.CreateSession(&copilot.SessionConfig{
		Model:     model,
		Streaming: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create summary session: %w", err)
	}
	defer func() {
		if err := session.Destroy(); err != nil {
			slog.Error("Failed to destroy summary session", slog.String("error", err.Error()))
		}
	}()

	// Set up event handler
	done := make(chan error, 1)

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				fmt.Print(formatSummaryOutput(*event.Data.DeltaContent))
			}

		case "assistant.reasoning_delta":
			if event.Data.DeltaContent != nil {
				fmt.Print(formatCopilotDim(*event.Data.DeltaContent))
			}

		case "assistant.message":
			// Print final message in yellow for summary
			if event.Data.Content != nil {
				fmt.Println(formatSummaryOutput(*event.Data.Content))
				slog.Debug("Summary response", slog.String("content", *event.Data.Content))
			}

		case "assistant.reasoning":
			// Print reasoning in dimmed style
			if event.Data.Content != nil {
				fmt.Println(formatCopilotDim(*event.Data.Content))
				slog.Debug("Summary reasoning", slog.String("content", *event.Data.Content))
			}

		case "session.idle":
			slog.Info("Summary session completed")
			done <- nil

		case "session.error":
			errMsg := "summary session error"
			if event.Data.Error != nil {
				errMsg = fmt.Sprintf("%s: %v", errMsg, event.Data.Error)
			}
			slog.Error("Summary session error", slog.String("error", errMsg))
			done <- fmt.Errorf("%s", errMsg)
		}
	})

	// Build summary prompt
	summaryPrompt := buildSummaryPrompt(outputs)

	slog.Info("Sending summary prompt to Copilot")

	_, err = session.Send(copilot.MessageOptions{
		Prompt: summaryPrompt,
	})
	if err != nil {
		return fmt.Errorf("failed to send summary message: %w", err)
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		fmt.Println() // Add newline after streaming output
		return nil

	case <-time.After(10 * time.Minute):
		return fmt.Errorf("summary session timed out after 10 minutes")

	case <-ctx.Done():
		return fmt.Errorf("summary session cancelled: %w", ctx.Err())
	}
}

// buildSummaryPrompt creates the prompt for the summary session
func buildSummaryPrompt(outputs []ChunkOutput) string {
	var prompt strings.Builder

	prompt.WriteString("# Summary Task\n\n")
	prompt.WriteString("You have just processed multiple chunks of changes for a web project using Vanilla Framework.\n")
	prompt.WriteString("Please provide a comprehensive summary of all the work completed.\n\n")

	prompt.WriteString("## Summary Requirements\n\n")
	prompt.WriteString("Please provide:\n\n")
	prompt.WriteString("1. **Overview**: Brief description of what was accomplished across all chunks\n")
	prompt.WriteString("2. **Files Modified**: List of files that were created or modified\n")
	prompt.WriteString("3. **Patterns Used**: Which Vanilla Framework patterns were implemented (Hero, Equal Heights, etc.)\n")
	prompt.WriteString("4. **Key Changes**: Highlight significant changes or additions\n")
	prompt.WriteString("5. **Potential Issues**: Any problems encountered or warnings to note\n")
	prompt.WriteString("6. **Next Steps**: Recommended actions before creating a PR\n\n")
	prompt.WriteString("Keep the summary concise but comprehensive. Focus on actionable information.\n\n")

	prompt.WriteString("## Chunks Processed\n\n")

	for _, output := range outputs {
		fmt.Fprintf(&prompt, "### Chunk %d\n\n", output.ChunkNumber)
		fmt.Fprintf(&prompt, "**Duration**: %s\n\n", output.Duration.Round(time.Millisecond))
		prompt.WriteString("**Output**:\n```\n")
		prompt.WriteString(output.Output)
		prompt.WriteString("\n```\n\n")
	}

	return prompt.String()
}
