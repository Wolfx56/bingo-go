# Bingo

A self-hosted bingo app: build custom cards (3×3 to 6×6), check cells off over time, and every
validation is saved to SQLite right away, along with its history.

- **Backend**: Go (standard library) + SQLite through `modernc.org/sqlite` (pure Go, no CGO).
- **Front end**: Vue 3 + Vuetify 4, built with Vite and **embedded in the Go binary**.
- **Deployment**: one binary, one database file. No authentication.

The UI is in French. Code, comments, logs, API routes and page URLs are in English.

## Features

- Several bingos side by side, with progress and a grid preview on the home page.
- Card creation: title, description, size, cells typed one by one or pasted as a list, random
  shuffle.
- Cell validation (with an optional note), saved the moment it is validated. A validation can be
  undone.
- Detection of completed rows, columns and diagonals ("Bingo!") and of a full card.
- Per-bingo and global history (validations, undos, completed lines…).
- Archiving (read-only), "replay" (a fresh copy of a card, shuffled or not), deletion.
- Light / dark theme, mobile friendly, nothing loaded from the Internet.

## Requirements

- Go ≥ 1.26
- Node.js ≥ 22 and npm (only to build the front end)

## Build and run

```bash
make build   # builds the front end (web/dist), then the ./bingo binary
./bingo      # http://localhost:8080
```

The front end is embedded at compile time, so it must always be built **before** the Go binary;
`make build` takes care of that.

### Configuration

| Flag    | Env variable | Default    | Purpose                                                     |
| ------- | ------------ | ---------- | ----------------------------------------------------------- |
| `-addr` | `BINGO_ADDR` | `:8080`    | Listen address (`127.0.0.1:8080` to stay on this machine)   |
| `-db`   | `BINGO_DB`   | `bingo.db` | Path to the SQLite file (created on first start)            |

A command-line flag takes precedence over its environment variable.

> The app has no authentication: anyone who can reach the port can change everything.
> Keep it on the local network, or put it behind a reverse proxy that handles access.

## Development

Two terminals:

```bash
go run . -addr 127.0.0.1:8080 -db bingo-dev.db   # API
```

```bash
cd web && npm install && npm run dev             # front end with hot reload (port 5173)
```

The Vite dev server proxies `/api` to `localhost:8080`. `go run .` needs `web/dist` to exist
(run `make web` once first).

```bash
make test    # go vet + Go tests + TypeScript type check
```

## Deploying with systemd

An example unit is provided in [deploy/bingo.service](deploy/bingo.service).

```bash
make linux   # build/bingo-linux-amd64 and build/bingo-linux-arm64
```

On the server (copy the matching binary and the `.service` file over):

```bash
sudo install -m 755 bingo-linux-amd64 /usr/local/bin/bingo
sudo install -m 644 bingo.service /etc/systemd/system/bingo.service
sudo systemctl daemon-reload
sudo systemctl enable --now bingo
journalctl -u bingo -f
```

The unit uses `DynamicUser` and `StateDirectory`: systemd creates a dedicated user and the
`/var/lib/bingo` directory, which holds the database. Change the listen address with
`Environment=BINGO_ADDR=…` in the unit.

### Backup

All data lives in the SQLite file (WAL mode). For a consistent copy while the service is running:

```bash
sudo sqlite3 /var/lib/bingo/bingo.db ".backup '/path/to/backup/bingo.db'"
```

Or stop the service and copy `bingo.db`.

## Pages

| Path               | Page                          |
| ------------------ | ----------------------------- |
| `/`                | Bingo list (active, archived) |
| `/new`             | Create a bingo                |
| `/bingos/{id}`     | Play a bingo                  |
| `/bingos/{id}/edit`| Edit a bingo                  |
| `/history`         | Global history                |

## API

All routes live under `/api` and exchange JSON. Errors look like `{"error": "message"}`; the
message is shown to the user as-is, so it is in French.

| Method   | Route                                     | Purpose                                             |
| -------- | ----------------------------------------- | --------------------------------------------------- |
| `GET`    | `/bingos`                                 | List bingos (with progress)                         |
| `POST`   | `/bingos`                                 | Create a bingo `{title, description, size, cells[]}` |
| `GET`    | `/bingos/{id}`                            | A bingo and its cells                               |
| `PUT`    | `/bingos/{id}`                            | Update title, description and labels                |
| `DELETE` | `/bingos/{id}`                            | Delete the bingo and its history                    |
| `POST`   | `/bingos/{id}/archive`                    | Archive (read-only)                                 |
| `DELETE` | `/bingos/{id}/archive`                    | Unarchive                                           |
| `POST`   | `/bingos/{id}/duplicate`                  | Fresh copy `{title, shuffle}`                       |
| `PUT`    | `/bingos/{id}/cells/{cellId}/validation`  | Validate a cell `{note}`                            |
| `DELETE` | `/bingos/{id}/cells/{cellId}/validation`  | Undo the validation                                 |
| `GET`    | `/bingos/{id}/history`                    | History of one bingo                                |
| `GET`    | `/history`                                | Global history (`?limit=`, 1000 max)                |

## Layout

```
main.go               entry point: flags, HTTP server, graceful shutdown
internal/store/       SQLite: schema, migrations, business rules, tests
internal/api/         JSON routes + embedded front end (SPA) serving
web/                  Vue 3 + Vuetify front end (web/dist is embedded by web/embed.go)
deploy/bingo.service  example systemd unit
```
