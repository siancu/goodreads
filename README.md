# Goodreads CLI (Go)

A command-line interface for Goodreads, written in Go.

Since Goodreads deprecated their public API in December 2020, this tool authenticates via email/password and saves session cookies locally. This is a Go port of the [Python version](https://github.com/siancu/goodreads).

## Installation

### Download a binary (recommended)

Grab the latest release for your platform from the [Releases page](https://github.com/siancu/goodreads-go/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/siancu/goodreads-go/releases/latest/download/goodreads-go_Darwin_arm64.tar.gz | tar xz
sudo mv goodreads /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/siancu/goodreads-go/releases/latest/download/goodreads-go_Darwin_amd64.tar.gz | tar xz
sudo mv goodreads /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/siancu/goodreads-go/releases/latest/download/goodreads-go_Linux_amd64.tar.gz | tar xz
sudo mv goodreads /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/siancu/goodreads-go/releases/latest/download/goodreads-go_Linux_arm64.tar.gz | tar xz
sudo mv goodreads /usr/local/bin/
```

**Windows**: download the `.zip` from the [Releases page](https://github.com/siancu/goodreads-go/releases) and add `goodreads.exe` to your PATH.

### Build from source

Requires [Go](https://go.dev/) 1.25+.

```bash
git clone git@github.com:siancu/goodreads-go.git
cd goodreads-go
go build -o goodreads .
```

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

### Authors

| Command | Description |
|---------|-------------|
| `./goodreads author search <query>` | Search for authors (`--limit N`, default 10) |
| `./goodreads author show <author-id>` | Show author bio, genres, website |
| `./goodreads author books <author-id>` | List books by an author (`--limit N`, default 20) |

### Users

| Command | Description |
|---------|-------------|
| `./goodreads user list` | List users you follow |
| `./goodreads user show <user-id>` | Show user profile |
| `./goodreads user shelves <user-id>` | List a user's shelves |
| `./goodreads user books <user-id>` | Show a user's books (`--shelf NAME`, `--limit N`) |
| `./goodreads user stats <user-id>` | Show a user's reading stats |

### Statistics

| Command | Description |
|---------|-------------|
| `./goodreads stats` | Show your reading statistics |
| `./goodreads stats <year>` | Show stats for a specific year |

## Cookie Expiration

Cookies expire periodically. If you get authentication errors, just run `./goodreads login` again.

## Disclaimer

This tool uses undocumented Goodreads web endpoints. It may break if Goodreads changes their website. Use at your own risk.

## License

MIT
