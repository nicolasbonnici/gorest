# migrate CLI

Standalone migration tool for GoREST applications.

```bash
go build -o migrate ./migrations/cmd/migrate/
```

## Connection

Priority order: `--dsn` flag → `DATABASE_URL` env → `--config <dir>/gorest.yaml` (default: `.`)

```bash
migrate <command> --config /var/apps/myapi   # read gorest.yaml
migrate <command> --dsn postgres://user:pass@host/db
```

## Commands

| Command | Description |
|---------|-------------|
| `migrate run` | Run all pending migrations |
| `migrate run <version> <source>` | Run one migration up |
| `migrate run <version> <source> --down` | Roll back one migration |
| `migrate status` | Show all migration records |
| `migrate pending` | List pending migrations |
| `migrate dry-run` | Preview what would run |
| `migrate new <name>` | Generate a new migration file |
| `migrate repair force <version> <source>` | Mark failed migration as applied (no SQL run) |
| `migrate repair retry <version> <source>` | Delete failed record so it re-runs on next startup |

## Examples

```bash
# See the current state (works with zero sources — reads DB directly)
migrate status --config .

# Run everything pending
migrate run --dsn $DATABASE_URL

# Roll back one specific migration
migrate run 20250121000001000 gorest-core-auth --down

# Generate a new migration
migrate new add_role_column --migration-dir ./migrations/

# Fix a dirty database (failed migration blocks startup)
migrate repair retry 20250121000001000 gorest-core-auth --config /var/apps/myapi
```

## Embedding in your app

Use `migrations/cli.Runner` to wire the CLI into your app's own `cmd/migrate` with full source knowledge (enables `pending`, `dry-run`, and targeted `run`):

```go
m := migrations.NewMigrator(db, appMigrations, pluginMigrations)
cli.New(m, db, "./migrations").Run(os.Args[1:])
```
