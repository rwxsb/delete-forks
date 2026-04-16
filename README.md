# delete-forks

A terminal UI for listing and bulk-deleting your GitHub forks.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/github/license/rwxsb/delete-forks)
![Release](https://img.shields.io/github/v/release/rwxsb/delete-forks)

## Features

- Lists all your forked repositories
- Select individual forks or select all at once
- Confirmation prompt before deletion
- Scrollable UI for large lists
- Progress tracking during deletion
- Handles SAML-protected org repos gracefully
- Auth via `gh` CLI — no tokens to manage

## Prerequisites

- [GitHub CLI (`gh`)](https://cli.github.com) installed and authenticated

## Install

### From release

Download a binary from the [releases page](https://github.com/rwxsb/delete-forks/releases).

### From source

```bash
go install github.com/rwxsb/delete-forks@latest
```

### Build locally

```bash
git clone https://github.com/rwxsb/delete-forks.git
cd delete-forks
go build -o delete-forks .
```

## Usage

```bash
./delete-forks
```

If you're not logged in to `gh`, it will launch `gh auth login` automatically and request the `delete_repo` scope.

## Controls

| Key | Action |
|-----|--------|
| `↑` / `k` | Navigate up |
| `↓` / `j` | Navigate down |
| `Space` | Toggle selection |
| `a` | Select / deselect all |
| `d` / `Enter` | Proceed to delete |
| `y` / `n` | Confirm / cancel deletion |
| `q` | Quit |

## Running tests

```bash
# Unit tests (no auth required)
go test ./tui/... -v

# Integration tests (requires gh auth)
go test ./github/... -v

# All tests
go test ./... -v
```
