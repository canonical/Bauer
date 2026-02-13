# Bauer

A proof-of-concept Go application that extracts document content, suggestions (proposed edits), and comments from Google Docs using the Google Docs API and Google Drive API.

## Installation

### [Snap](https://snapcraft.io/bauer)

```
sudo snap install bauer
```

### Homebrew

First time installation

```
brew install britneywwc/bauer/bauer
```

Upgrade to a newer version or later

```
brew update
brew upgrade bauer
```

N.B. You need to install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli) which is used by Bauer.

## Configuration

1. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)
2. Copy `.env.example` to `.env`
3. Fill `.env` with Google Cloud credentials (see [Generating Google Cloud credentials](https://developers.google.com/workspace/guides/create-credentials))
4. Set `API_SECRET` in `.env` for API basic auth
5. Share copy document with the service account from `GOOGLE_CLIENT_EMAIL`

If you already have `credentials.json`, migrate it with:

```bash
python3 scripts/migrate_credentials_to_env.py --input credentials.json --output .env
```

## Usage

1. Install bauer using the instructions above
2. Check that `copilot` and `bauer` are installed
3. Get document ID from Google Document & share the document with the service account
4. Run Bauer

```bash
bauer --doc-id <your-document-id>
```

6. Optional parameters

| Flag             | Type   | Default           | Description                                                                  |
| ---------------- | ------ | ----------------- | ---------------------------------------------------------------------------- |
| `--chunk-size`   | int    | `1`               | Total number of chunks to create (default: 1, or 5 if --page-refresh is set) |
| `--dry-run`      | bool   | `false`           | Run extraction and planning only; skip Copilot execution and PR creation     |
| `--output-dir`   | string | `bauer-output`    | Output directory for generated files                                         |
| `--model`        | string | `gpt-5-mini-high` | Copilot model to use for code generation                                     |
| `--page-refresh` | bool   | `false`           | Whether this is a page refresh, or the default copy update                   |
| `--target-repo`  | string | current directory | Path to target repository where tasks should be executed                     |
### Examples

#### Basic run

```bash
bauer --doc-id <your-document-id>
```

#### Dry run (test without executing changes)

```bash
bauer --doc-id <your-document-id> \
        --dry-run
```

#### Custom chunk size and output directory

```bash
bauer --doc-id <your-document-id> \
        --chunk-size 5 \
        --output-dir ./results
```

#### Specify model

```bash
bauer --doc-id <your-document-id> \
        --model "claude-sonnet-4.5"
```

#### Run on a different repository
```bash
bauer --doc-id <your-document-id> \
        --target-repo ../my-other-repo
```

### Page refresh

```bash
bauer --doc-id <your-document-id> \
        --page-refresh
```

## API usage

The API server exposes a small HTTP surface for submitting jobs and checking health. Jobs run asynchronously and write outputs to `base-output-dir/<request-id>`.

### Run the API server

From Repository root:

```bash
task build-api
./bauer-api --config config.json
```

The API requires HTTP basic auth for all endpoints except `/api/v1/health`.
Use username `bauer` and password from `API_SECRET`.

### Endpoints

#### POST /api/v1/job

Submit a job for a Google Doc.

Request body:

```json
{
  "doc_id": "<google-doc-id>",
  "chunk_size": 1,
  "page_refresh": false
}
```

Notes:

- `chunk_size` defaults to 1 if omitted.
- When `page_refresh` is true, the default chunk size becomes 5.

Responses:

- `202 Accepted` with body `{"code":202}` when the job is accepted.
- `400 Bad Request` for invalid JSON.

Example:

```bash
curl -X POST http://localhost:8090/api/v1/job \
        -u "bauer:${API_SECRET}" \
        -H 'Content-Type: application/json' \
        -d '{"doc_id":"<google-doc-id>","chunk_size":2,"page_refresh":false}'
```

#### GET /api/v1/health

Simple health check.

Example:

```bash
curl http://localhost:8090/api/v1/health
```

## Local development

### Prerequisites

1. Install [`go`](https://golang.org/dl/)
2. Install [`task`](https://taskfile.dev/docs/installation)
3. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)

## Steps

1. Modify the [Taskfile](./Taskfile.yml) with your document ID for convenience
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
