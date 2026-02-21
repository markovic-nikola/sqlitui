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

# Print version
sqlitui --version

# Update to the latest release
sqlitui --update
```

Supported file extensions: `.db`, `.sqlite`, `.sqlite3`

## Update

```bash
sqlitui --update
```

This checks GitHub for a newer release, downloads it, verifies the SHA256 checksum, and replaces the binary in place.

If installed to a system directory (e.g. `/usr/local/bin`), you may need:

```bash
sudo sqlitui --update
```

Alternatively, re-run the install script to get the latest version.

## License

[MIT](LICENSE)
