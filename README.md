# vine

Task tracking for AI agents. Pick off tasks like grapes on a vine.

## Install

Ensure `~/go/bin` is in your PATH by adding this to your `~/.zshrc` (or `~/.bashrc`):

```sh
export PATH="$HOME/go/bin:$PATH"
```

### As a dev

From the repo root:

```sh
go install .
```

There is no hot reloading. Go compiles to a binary, so you need to re-run `go install .` after every change to update the `vine` binary in your PATH.

## Usage

```sh
vine init                           # setup a new project
vine create "Fix login bug" -t bug  # create a task
vine status                         # see task summary
vine status --detailed              # with type breakdown
```

## Publishing

Before publishing for `go install` to work remotely:

1. Choose a hosting URL (e.g., `github.com/yourorg/vine`)
2. Update the module path in `go.mod`
3. Find and replace `"vine/` with `"github.com/yourorg/vine/` in all `.go` imports
4. Run `go mod tidy && go build ./...` to verify
5. Update the install command below:

```sh
go install github.com/yourorg/vine@latest
```
