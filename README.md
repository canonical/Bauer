# Bauer

A proof-of-concept Go application that extracts document content, suggestions (proposed edits), and comments from Google Docs using the Google Docs API and Google Drive API.


## Installation

### [Snap](https://snapcraft.io/bauer)
```
sudo snap install bauer
```

### Homebrew
```
brew install britneywwc/bauer
```

N.B. You need to install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli) Copilot CLI which is used by Bauer.

## Configuration
1. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)
2. Create `credentials.json` file and copy the structure from the [default file](https://github.com/muhammadbassiony/Bauer/blob/main/credentials.json)
3. Get credentials from Google Cloud service or Bitwarden (internally)
4. Fill up `credentials.json` with Google Cloud credentials (see [Generating Google Cloud credentials](https://developers.google.com/workspace/guides/create-credentials)).
5. Share copy document with service account

## Usage
1. Install bauer using the instructions above
2. Check that `bauer` is installed
```
bauer
```
3. Get document ID from Google Document & share the document with the service account
4. Run Bauer
```bash
bauer --doc-id <your-document-id> --credentials ./credentials.json
```

6. Optional parameters

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--chunk-size` | int | `1` | Total number of chunks to create (default: 1, or 5 if --page-refresh is set) |
| `--dry-run` | bool | `false` | Run extraction and planning only; skip Copilot execution and PR creation |
| `--output-dir` | string | `bauer-output` | Output directory for generated files |
| `--model` | string | `gpt-5-mini-high` | Copilot model to use for code generation |
| `--page-refresh` | bool | `false` | Whether this is a page refresh, or the default copy update |

### Examples

#### Basic run
```bash
bauer --doc-id <your-document-id> --credentials ./credentials.json
```

#### Dry run (test without executing changes)
```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --dry-run
```

#### Custom chunk size and output directory
```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --chunk-size 5 \
        --output-dir ./results
```

#### Specify model
```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --model "claude-sonnet-4.5"
```
### Page refresh
```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --page-refresh
```

## Local development
### Prerequisites
1. Install [`go`](https://golang.org/dl/)
2. Install [`task`](https://taskfile.dev/docs/installation)
3. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)

## Steps
1. Modify the [Taskfile](./Taskfile.yml) with your document ID and credentials path for convenience
2. Run the project with task
```
task run
```

## Documentation
For more information refer to [`ARCHITECTURE.md`](/docs/ARCHITECTURE.md)

## Future improvements

### Short term
- Automatically open PR with changes applied to the document using Google Docs API
- Improve prompt templates for better results (this requires a lot of trial and error)

for code improvements, you can also refer to our [todo](./todo.txt) list

### Long term

On the long term, BAUer should evolve into a full-fledged API service, with the following features:
- Automatic Jira ticket hooks to trigger workflows
- Unified service account with domain wide delegation
- Calling LLMs - with varying implementation complexity - via: 
        - calling LLM APIs directly
        - spinning up ephemeral Copilot CLI instances
        - self-hosted LLMs (can use open source models such as Llama, openAI OSS, deepseek, etc)
- Automatic PR creations and reviewer assignments
