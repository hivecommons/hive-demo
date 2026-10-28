# bookmarks API

A small JSON API for saving links.

## Running

```sh
PORT=8080 go run .
```

Bookmarks are kept in memory and lost when the server stops.

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Health check |
| `GET` | `/bookmarks?page=1` | List bookmarks, 20 per page |
| `POST` | `/bookmarks` | Create a bookmark |
| `GET` | `/bookmarks/{id}` | Get one bookmark |
| `DELETE` | `/bookmarks/{id}` | Delete a bookmark |

A bookmark:

```json
{
  "url": "https://go.dev",
  "title": "The Go programming language",
  "tags": ["go"]
}
```

`url` must be an absolute `http` or `https` URL, and `title` is required.

## Development

```sh
make test
make vet
```
