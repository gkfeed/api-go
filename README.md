# GKFeed API

GKFeed is a small FastAPI application that stores feed subscriptions and items in SQLite and can expose a user's items as JSON or RSS. Interactive OpenAPI documentation is available at `/docs` while the server is running.

## Requirements

- Python 3.12 or newer
- [uv](https://docs.astral.sh/uv/)
- A SQLite database with the GKFeed schema

## Development

Install the locked dependencies and start the development server:

```sh
uv sync
make dev
```

The API listens on <http://localhost:8086>. Passwords are submitted only to the login endpoint. It returns a short-lived opaque access token and a rotating refresh token.

Use HTTPS in deployment (typically through a TLS-terminating reverse proxy), since login credentials and bearer tokens must not travel over plaintext connections.

Run the local quality checks with:

```sh
make check
```

`make format` applies Ruff's safe lint fixes and formatting.

## Configuration

Configuration is read from environment variables at startup:

| Variable | Default | Description |
| --- | --- | --- |
| `GKFEED_ADDRESS` | `:8086` | HTTP server listen address |
| `GKFEED_DB_PATH` | `data/db.sqlite` | SQLite path relative to the repository root |
| `GKFEED_ALLOWED_ORIGINS` | Localhost development origins | Comma-separated CORS origins |

## Docker

```sh
docker compose up --build -d
```

Docker Compose mounts `~/.local/share/gkfeed/data` at `/data` and configures the API to use `/data/db.sqlite`.

## API routes

| Method | Route | Authentication | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v2/auth/login` | None | Exchange JSON `username` and `password` for access and refresh tokens |
| `POST` | `/api/v2/auth/refresh` | Refresh token | Rotate the refresh token and issue a new access token |
| `POST` | `/api/v2/auth/logout` | Refresh token | Revoke the refresh-token family and its access tokens |
| `POST` | `/api/v2/auth/logout-all` | Bearer | Revoke every session belonging to the user |
| `GET` | `/api/v2/list` | Bearer | List the user's feeds |
| `GET` | `/api/v2/feed` | Bearer | Return the user's RSS feed |
| `POST` | `/api/v2/add` | Bearer | Add a feed |
| `POST` | `/api/v2/add_lazy` | Bearer | Add a feed inferred from its URL |
| `DELETE` | `/api/v2/delete?id=<id>` | Bearer | Delete a feed |
| `POST` | `/api/v2/add_deleted_items` | Bearer | Hide items for the user |
| `GET` | `/api/v2/get_items` | Bearer | Return cursor-paginated items |
| `GET` | `/api/v2/opengraph?url=<url>` | Bearer | Return Open Graph metadata for a public web page |
| `GET` | `/api/v2/item?id=<id>` | None | Return an item and its feed |

Log in and use the returned access token like this:

```sh
curl -sS -X POST http://localhost:8086/api/v2/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"reader","password":"your password"}'

curl -H 'Authorization: Bearer <token>' http://localhost:8086/api/v2/list
```

The response contains `access_token`, `refresh_token`, `token_type`, `expires_in`, and `refresh_expires_in`. Access tokens expire after 30 minutes and are held in memory. Refresh tokens have a rotating 90-day idle window and are exchanged at `/api/v2/auth/refresh`. Clients must allow only one refresh at a time and replace both stored tokens atomically. Reusing a token that has already been rotated revokes its entire token family.

Refresh-token SHA-256 digests and their revocation state are persisted in the automatically created `auth_refresh_tokens` SQLite table. Raw refresh tokens are never stored. An API restart invalidates access tokens, but a valid refresh token can obtain a new one.

## Password storage

The `users.password` column must contain a self-describing Argon2id hash, never a plaintext password. Generate a database-ready hash by sending the password on standard input:

```sh
read -rsp 'Password: ' PASSWORD
printf '%s' "$PASSWORD" | uv run python -m hash_password
unset PASSWORD
```

Each invocation uses a cryptographically random 16-byte salt and the OWASP minimum Argon2id settings `m=19456`, `t=2`, and `p=1`. Existing plaintext database values must be replaced with generated hashes before users can log in.
