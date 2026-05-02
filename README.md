# var-cli

A terminal UI (TUI) application for tracking workday time entries with the Elaniin VAR backend API.

## Features

- 🔐 Bearer token authentication
- 📊 Weekly time entry view with last 7 days (toggle to 2 weeks)
- 📝 Step-by-step form to create new time entries
- 🏷️ Smart project/tag suggestions based on recent usage
- 📋 Clipboard paste support (Ctrl+V) for API tokens

## Installation

### Quick install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/json-nan/var-cli/main/install.sh | sh
```

Or with a custom install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/json-nan/var-cli/main/install.sh | INSTALL_DIR=$HOME/bin sh
```

### Manual download

Download the latest release for your platform from the [releases page](https://github.com/json-nan/var-cli/releases).

### Verify installation

```bash
var-cli --version
```

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

### New entry form

1. **Date** — defaults to today (`YYYY-MM-DD`)
2. **Description** — what you worked on
3. **Project** — pick from frequently-used projects first
4. **Tags** — multi-select with Space, frequently-used first
5. **Time** — minutes (e.g. `60`, `480`)
6. **Billable** — toggle Yes/No

Press `Esc` at any step to cancel.

### Getting your API Token

1. Log in to [VAR](https://var.elaniin.com)
2. Open DevTools → Network tab
3. Find a `/projects` request
4. Copy the `Bearer` token from the `Authorization` header
5. Paste it into var-cli with **Ctrl+V**

## Development

```bash
go run main.go
```

## License

MIT
