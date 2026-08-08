# GKFeed API

GKFeed is a small Go HTTP API that stores feed subscriptions and items in PostgreSQL and can expose a user's items as JSON or RSS.

## Requirements

- Go 1.24 or newer
- PostgreSQL 14 or newer

## Development

Start PostgreSQL, configure a JWT secret, and run the API:

```sh
export GKFEED_JWT_SECRET="$(openssl rand -base64 32)"
docker compose up -d postgres
make dev
```

The default development connection is
`postgres://gkfeed:gkfeed@localhost:5432/gkfeed?sslmode=disable`. Override it
with `GKFEED_DATABASE_URL` when PostgreSQL runs elsewhere.

The API listens on <http://localhost:8086>. Most routes require HTTP Basic authentication using credentials stored in the `users` table. The `hashed_password` column stores PHC-formatted Argon2id hashes; plaintext passwords from an existing database are converted automatically during startup migration.

Run the local quality checks with:

```sh
make check
```

`make lint` runs `go vet` and the same `gopls` diagnostics used by Go editor
integrations. Install `gopls` first with `make install-lint`.

## Configuration

Configuration is read from environment variables at startup:

| Variable | Default | Description |
| --- | --- | --- |
| `GKFEED_ADDRESS` | `:8086` | HTTP server listen address |
| `GKFEED_DATABASE_URL` | Local Docker PostgreSQL URL | PostgreSQL connection URL |
| `GKFEED_ALLOWED_ORIGINS` | Localhost development origins | Comma-separated CORS origins |
| `GKFEED_JWT_SECRET` | none (required) | Cryptographically random JWT signing secret of at least 32 bytes |

## Docker

```sh
export GKFEED_JWT_SECRET="$(openssl rand -base64 32)"
docker compose up --build -d
```

Docker Compose starts PostgreSQL, waits for it to become healthy, and persists
its data in the `postgres-data` volume. Set `GKFEED_POSTGRES_PASSWORD` to change
the development database password.

Existing SQLite databases are not migrated automatically. Export and transform
their data before switching a production deployment to this version.

## PostgreSQL integration tests

Database tests use a separate, isolated schema and run when
`GKFEED_TEST_DATABASE_URL` is set:

```sh
export GKFEED_TEST_DATABASE_URL="postgres://gkfeed:gkfeed@localhost:5432/gkfeed?sslmode=disable"
make test
```

Without that variable, PostgreSQL integration tests are skipped and the rest of
the test suite still runs.

## API routes

Swagger UI is available at `/api/swagger/index.html`.

| Method | Route | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/list` | Basic | List the user's feeds |
| `GET` | `/api/v1/feed_types` | None | List feed types supported by the parser |
| `GET` | `/api/v1/feed` | Basic | Return the user's RSS feed |
| `POST` | `/api/v1/add` | Basic | Add a feed |
| `POST` | `/api/v1/add_lazy` | Basic | Add a feed inferred from its URL |
| `DELETE` | `/api/v1/delete?id=<id>` | Basic | Delete a feed |
| `POST` | `/api/v1/add_deleted_items` | Basic | Hide items for the user |
| `GET` | `/api/v1/get_items` | Basic | Return cursor-paginated items |
| `GET` | `/api/v1/item?id=<id>` | Basic or Bearer | Return the authenticated user's item and its feed |
| `GET` | `/api/v1/auth/me` | Basic or Bearer | Return the authenticated user |
