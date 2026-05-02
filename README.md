# var-cli

A terminal UI (TUI) application for tracking workday time entries with the Elaniin VAR backend API.

## Features

- 🔐 Bearer token authentication
- 📊 Weekly time entry view with last 7 days (toggle to 2 weeks)
- 📝 Step-by-step form to create new time entries
- 🏷️ Smart project/tag suggestions based on recent usage
- 📋 Clipboard paste support (Ctrl+V) for API tokens

## Installation

Download the latest release for your platform from the [releases page](https://github.com/elaniin/var-cli/releases).

## Usage

```bash
var-cli
```

### Keys

| Key | Action |
|-----|--------|
| `n` | New time entry |
| `r` | Refresh entries |
| `a` | Toggle 7 days / 2 weeks view |
| `q` | Quit |

### Getting your API Token

1. Log in to [VAR](https://var.elaniin.com)
2. Open DevTools → Network tab
3. Find a `/projects` request
4. Copy the `Bearer` token from the `Authorization` header

## Development

```bash
go run main.go
```

## License

MIT
