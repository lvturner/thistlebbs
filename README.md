# Thistle BBS

A telnet BBS server written in Go, focused on message boards. ThistleBBS is a
modern interpretation of old BBS systems — it is not intended to be historically
or "scene" accurate. Its primary use case is
[bbs.futurepast.online](https://bbs.futurepast.online).

Connect with any telnet client.

## Features

- Telnet protocol: NAWS (window size), TTYPE, SGA, BINARY negotiation
- ANSI colour + ASCII block-art banner
- Account creation (username, password, display name, location, bio)
- Single message board with threaded discussions
- Multi-line message editor (end with `.` on its own line)
- Paginated thread listing and per-message post viewing (jump to any post by number)
- Profile view and edit
- Outbound RLOGIN door: the [G]ames menu item connects to the gOLD mINE
  community door server (see the `-games` / `-games-tag` flags)
- SQLite persistence (WAL mode, busy timeout)
- Template-driven menus loaded from `data/menus/`

## Build

Requires Go 1.23+ and a C compiler (for the SQLite driver).

```sh
go build -o thistlebbs .
```

Or run directly:

```sh
go run . -listen :1997 -db data/thistlebbs.db -name "Thistle BBS"
```

## Usage

```
thistlebbs [flags]

  -banner string   path to banner file (empty = built-in) (default "data/banner.txt")
  -db string       path to the SQLite database file (default "data/thistlebbs.db")
  -games string      games door address (gOLD mINE), host:port (empty = disabled) (default "goldminedoors.com:2513")
  -games-tag string 3-character BBS tag sent to the games door as [TAG]<username> (empty = disabled) (default "FPO")
  -listen string   address to listen on (host:port) (default ":1997")
  -menu-dir string path to menu templates directory (empty = no templates) (default "data/menus")
  -name string     display name of the system (default "Thistle BBS")
  -version         print version and exit
```

Connect:

```sh
telnet localhost 1997
```

## Controls

Menus respond to a single keypress (no Enter needed). Free-form input (thread
numbers, usernames, passwords, message text, etc.) is line-based: type your
answer and press Enter.

| Key       | Action                |
|-----------|-----------------------|
| Ctrl-C    | Abort current input   |
| Ctrl-D    | Hang up / end input   |
| Ctrl-U    | Kill current line     |

## Layout

```
main.go                             entrypoint
data/banner.txt                     banner artwork (see data/BANNER_TAGS.md)
data/menus/                         template-driven menu screens
internal/telnet/                    IAC parser, line encoder, negotiation policy
internal/ansi/                      SGR colours, text layout, block-letter banner
internal/store/                     SQLite schema, users/threads/posts CRUD
internal/server/                    accept loop, session state machine, templates
```

## Tests

```sh
go test ./...
go vet ./...
```