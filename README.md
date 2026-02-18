```
 ▄▄▄▄  ▄▄▄  ▄▄    ▄▄ ▄▄▄▄▄▄ ▄▄ ▄▄ ▄▄ 
███▄▄ ██▀██ ██    ██   ██   ██ ██ ██ 
▄▄██▀ ▀███▀ ██▄▄▄ ██   ██   ▀███▀ ██ 
         ▀▀                          
```

A terminal UI for browsing and querying SQLite databases.

## Install

### Quick install (Linux / macOS)

```bash
curl -sSfL https://raw.githubusercontent.com/markovic-nikola/sqlitui/main/install.sh | sh
```

Downloads the latest release, verifies the SHA256 checksum, and installs to `/usr/local/bin`. Set `INSTALL_DIR` to change the location:

```bash
curl -sSfL https://raw.githubusercontent.com/markovic-nikola/sqlitui/main/install.sh | INSTALL_DIR=~/.local/bin sh
```

### From source

```bash
go install github.com/markovic-nikola/sqlitui@latest
```

### From releases

Download a prebuilt binary from the [releases page](https://github.com/markovic-nikola/sqlitui/releases).

## Usage

```bash
# Open a database directly
sqlitui <database.db>

# Or launch and enter the path interactively
sqlitui
```

Supported file extensions: `.db`, `.sqlite`, `.sqlite3`

## License

[MIT](LICENSE)
