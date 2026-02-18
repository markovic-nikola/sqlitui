```
 ▄▄▄▄  ▄▄▄  ▄▄    ▄▄ ▄▄▄▄▄▄ ▄▄ ▄▄ ▄▄ 
███▄▄ ██▀██ ██    ██   ██   ██ ██ ██ 
▄▄██▀ ▀███▀ ██▄▄▄ ██   ██   ▀███▀ ██ 
         ▀▀                          
```

A terminal UI for browsing and querying SQLite databases.

## Install

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
