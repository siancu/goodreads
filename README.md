# Goodreads CLI (Go)

A command-line interface for Goodreads, written in Go.

Since Goodreads deprecated their public API in December 2020, this tool authenticates via email/password and saves session cookies locally. This is a Go port of the [Python version](https://github.com/siancu/goodreads).

## Prerequisites

- [Go](https://go.dev/) 1.25+
- A Goodreads account

## Installation

1. Clone the repository:
   ```bash
   git clone git@github.com:siancu/goodreads-go.git
   cd goodreads-go
   ```

2. Build the binary:
   ```bash
   go build -o goodreads .
   ```

3. Log in (see [Authentication](#authentication))

## Authentication

### Interactive login

```bash
./goodreads login
```

You'll be prompted for your email and password. Cookies are saved to `~/.goodreads-cookies.json`.

### Non-interactive login

For scripts and automation, set environment variables:

```bash
export GOODREADS_EMAIL="your@email.com"
export GOODREADS_PASSWORD="yourpassword"
./goodreads login
```

### Login flags

```bash
./goodreads login --email your@email.com --password yourpassword
./goodreads login -e your@email.com -p yourpassword
./goodreads login --debug  # Save intermediate HTML pages to /tmp for debugging
```

### Logout

```bash
./goodreads logout
```

## Usage

### Authentication

| Command | Description |
|---------|-------------|
| `./goodreads login` | Log in to Goodreads |
| `./goodreads logout` | Log out (remove saved cookies) |

### Shelves

| Command | Description |
|---------|-------------|
| `./goodreads shelf list` | List all your shelves |
| `./goodreads shelf show <name>` | Show books on a shelf |
| `./goodreads shelf add <name>` | Create a new shelf |
| `./goodreads shelf delete <name>` | Delete a shelf (`--force` to skip confirmation) |

### Books

| Command | Description |
|---------|-------------|
| `./goodreads book search <query>` | Search for books (`--limit N`, default 10) |
| `./goodreads book show <book-id>` | Show detailed book info |
| `./goodreads book add <book-id> <shelf>` | Add a book to a shelf |
| `./goodreads book remove <book-id> [shelf]` | Remove a book from a shelf (or all shelves) |
| `./goodreads book rate <book-id> <1-5>` | Rate a book |
| `./goodreads book similar <book-id>` | Find similar books (`--limit N`, `--show-lists`, `--list N`) |

## Cookie Expiration

Cookies expire periodically. If you get authentication errors, just run `./goodreads login` again.

## Disclaimer

This tool uses undocumented Goodreads web endpoints. It may break if Goodreads changes their website. Use at your own risk.

## License

MIT
