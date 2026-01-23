package copilotcli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
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

// ExecuteChunk processes a single chunk prompt using a Copilot session
func (c *Client) ExecuteChunk(ctx context.Context, chunkPath string, chunkNumber int, model string) error {
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
		return fmt.Errorf("failed to create session for chunk %d: %w", chunkNumber, err)
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

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		// TODO these 2 events should be only for debugging/verbose logging
		case "assistant.message_delta":
			// Stream incremental content as it comes (in color)
			if event.Data.DeltaContent != nil {
				fmt.Println(formatCopilotOutput(*event.Data.DeltaContent))
			}

		case "assistant.reasoning_delta":
			// Stream reasoning content (dimmed)
			if event.Data.DeltaContent != nil {
				fmt.Println(formatCopilotDim(*event.Data.DeltaContent))
			}

		case "assistant.message":
			if event.Data.Content != nil {
				slog.Debug("Assistant response",
					slog.Int("chunk", chunkNumber),
					slog.String("content", *event.Data.Content),
				)
			}

		case "assistant.reasoning":
			if event.Data.Content != nil {
				slog.Debug("Assistant reasoning response",
					slog.Int("chunk", chunkNumber),
					slog.String("content", *event.Data.Content),
				)
				// Print reasoning in dimmed style
				fmt.Println(formatCopilotDim(*event.Data.Content))
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
		return fmt.Errorf("failed to resolve chunk path: %w", err)
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
		return fmt.Errorf("failed to send message for chunk %d: %w", chunkNumber, err)
	}

	// Wait for completion with timeout
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		fmt.Println() // Add newline after streaming output
		return nil

	case <-time.After(15 * time.Minute):
		return fmt.Errorf("chunk %d timed out after 15 minutes", chunkNumber)

	case <-ctx.Done():
		return fmt.Errorf("chunk %d cancelled: %w", chunkNumber, ctx.Err())
	}
}
