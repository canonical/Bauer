# Workshop Tutorial

## What Workshop is

Workshop is Canonical's container-based development environment tool.

- It uses LXD to create an isolated dev environment for a project.
- The environment is defined in `workshop.yaml` at the repo root.
- Your repository is mounted into the container at `/project`.
- Tooling is added through SDKs, not by mutating your host machine.

For Bauer, the current workshop definition includes:

- `go` for the Go toolchain
- `vscode-remote` for VS Code remote access
- `opencode` for the optional OpenCode CLI flow

## How it works

Workshop has five core pieces:

1. `base`: the underlying OS image, currently `ubuntu@24.04`
2. `sdks`: reusable tool bundles layered on top of the base
3. `mounts`: persistent paths that survive rebuilds, such as caches
4. `actions`: named commands defined in `workshop.yaml`
5. lifecycle commands: `launch`, `refresh`, `start`, `stop`, `remove`

The important behavior is:

- `workshop launch` creates the workshop the first time
- `workshop start` restarts a stopped workshop
- `workshop refresh` applies `workshop.yaml` changes to an existing workshop
- `workshop remove` deletes the workshop so it can be rebuilt from scratch
- `workshop exec dev -- ...` runs one command in the workshop
- `workshop shell` opens an interactive shell in the workshop

## Daily cheat sheet

### First-time setup

```bash
workshop launch
workshop info
workshop tasks
```

### If the workshop already exists

If you see:

```text
error: cannot launch "dev": workshop exists
```

that means `launch` is the wrong command for the current state.

Use this rule:

- if the workshop is already `Ready`, just use it
- if it is `Stopped`, run `workshop start`
- if `workshop.yaml` changed, run `workshop refresh`
- if you want a clean rebuild, run `workshop remove && workshop launch`

This repo now includes an idempotent helper:

```bash
task workshop-up
```

That helper:

- launches the workshop if it does not exist
- starts it if it exists but is stopped
- leaves it alone if it is already ready

### Common Bauer commands

Run these from the host, not from inside the workshop:

```bash
task workshop-build
task workshop-test
task workshop-run-server CREDENTIALS=credentials.json
task workshop-shell
```

Equivalent raw Workshop commands are:

```bash
workshop exec dev -- bash -lc 'cd /project && go build -o bauer ./cmd/bauer && go build -o bauer-api ./cmd/app'
workshop exec dev -- bash -lc 'cd /project && go test --cover ./...'
workshop run dev -- run-api /project/credentials.json
workshop shell
```

### When to use `task` vs `workshop`

- Use `task ...` on the host for repeatable project commands
- Use `workshop ...` when you need lifecycle control, inspection, or ad hoc execution

`task` is not currently installed inside the workshop VM, so the cleanest setup is to keep Task on the host and make Task call into Workshop.

## VS Code remote setup

### What the `vscode-remote` SDK does

The Workshop `vscode-remote` SDK prepares the workshop so VS Code can attach over SSH.

In practice it does three things that matter here:

- configures SSH access to the workshop
- prepares the remote VS Code server location
- exposes connection instructions through `workshop tasks`

After launch, `workshop tasks` prints a hint like:

```text
VS Code → Open Remote Window → Connect to host → workshop@10.x.x.x
code --folder-uri vscode-remote://ssh-remote+workshop@10.x.x.x/project
```

### How Remote - SSH works

VS Code Remote - SSH is a local extension that connects to a remote host over SSH.

- VS Code UI stays local on your laptop
- the remote host runs the VS Code server and extensions
- terminals, language servers, builds, and debugging run on the remote host
- you open `/project` on the remote host, not the local checkout

That is why the extension is required here: the Workshop SDK prepares the remote host side, but the VS Code extension provides the client side.

### Recommended VS Code flow

1. Install `ms-vscode-remote.remote-ssh`
2. Run `task workshop-up`
3. Run `workshop tasks`
4. Copy the `workshop@...` target
5. In VS Code run `Remote-SSH: Connect to Host...`
6. Open `/project`

## Can this be used with Zed?

Yes, with an important caveat.

- Zed supports SSH-based remote development
- Zed uses its own remote server protocol
- Zed does not use the VS Code Remote - SSH extension
- Zed also does not use the Workshop `vscode-remote` SDK directly

So the answer is:

- you can likely connect Zed to the same workshop host over SSH
- but that is generic SSH remote development, not the Workshop VS Code SDK integration

In other words:

- VS Code path: Workshop `vscode-remote` SDK + Remote - SSH extension
- Zed path: plain SSH connection to the same host, with Zed deploying its own remote server

For Zed, the rough flow is:

1. Ensure the workshop is running
2. Get the `workshop@...` address from `workshop tasks`
3. In Zed, create a remote SSH connection to that host
4. Open `/project`

This was not validated end to end here, so treat it as supported in principle by Zed's SSH model, not as a Bauer-specific tested path.

## Issues hit during setup

### `workshop launch` is not idempotent

Problem:

- rerunning `workshop launch` after the workshop already exists fails with `workshop exists`

Mitigation:

- added `task workshop-up`
- documented when to use `start`, `refresh`, and `remove`

### `task` is not available inside the workshop

Problem:

- host `task` commands do not automatically exist inside the workshop environment

Mitigation:

- added host-side tasks that call `workshop exec` and `workshop run`
- no need to install `task` in the VM for normal Bauer workflows

### Bauer API requires explicit credentials

Problem:

- Bauer's current API startup path exits early if `--credentials` is not provided

Mitigation:

- `run-api` in `workshop.yaml` requires a credentials path argument
- `task workshop-run-server CREDENTIALS=credentials.json` now mirrors that requirement

### In-workshop Go downloads can fail on restricted networking

Problem:

- during validation, one workshop build attempt failed reaching `proxy.golang.org`

Mitigation:

- this is a network/proxy problem, not a Workshop definition problem
- if it recurs, configure proxy variables in the workshop environment or ensure the workshop has outbound network access

### Existing Bauer tests are not fully green

Problem:

- `go test ./...` currently fails in `internal/config` with `missing required field: type`

Mitigation:

- left this unchanged because it is an existing repo issue unrelated to the Workshop setup

## Practical guidance

- Use `task workshop-up` instead of rerunning `workshop launch`
- Use `task workshop-refresh` after editing `workshop.yaml`
- Use host-side Task commands to drive in-workshop builds and runs
- Use VS Code Remote - SSH if you want the path currently validated for Bauer
- Use Zed only as a generic SSH remote client, not as a consumer of the VS Code SDK