# user CLI

Manage users and their roles against a live GoREST database.

```bash
go build -o user ./cmd/user/
```

## Connection

Priority order: `--dsn` flag → `DATABASE_URL` env → `--config <dir>/gorest.yaml` (default: `.`)

```bash
user <command> --config /var/apps/myapi
user <command> --dsn postgres://user:pass@host/db
```

## Commands

### Users

| Command | Description |
|---------|-------------|
| `user list` | List all users with their roles and last assignment date |
| `user show <user-id>` | Show roles for a specific user |
| `user create` | Interactive prompt: first name, last name, email, password |
| `user password <user-id\|email>` | Interactive prompt to reset a user's password |
| `user promote <user-id> <role>` | Assign a role to a user |
| `user demote <user-id> <role>` | Remove a role from a user |

### Roles

| Command | Description |
|---------|-------------|
| `user roles list` | List all defined roles with description and parent |
| `user roles hierarchy` | Display the role inheritance tree |

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config <dir>` | `.` | Directory containing `gorest.yaml` |
| `--dsn <url>` | — | Raw database URL (overrides config) |
| `--output`, `-o` | `table` | Output format: `table` or `json` |
| `--actor <name>` | `cli` | Identity recorded in the audit log for promote/demote |

## Examples

```bash
# List all users
user list --config /var/apps/myapi

# Show one user's roles
user show 8f47cdb6-9f7e-214d-3fc7-83cfefaff433

# Create a new user (interactive)
user create --config /var/apps/myapi
# → First name: Nicolas
# → Last name: Bonnici
# → Email: nicolas@example.com
# → Password: (hidden)
# → Confirm password: (hidden)
# User created: 8f47cdb6-...

# Reset a password (accepts UUID or email)
user password nicolas@example.com
user password 8f47cdb6-9f7e-214d-3fc7-83cfefaff433
# → New password: (hidden)
# → Confirm password: (hidden)
# Password updated.

# Assign a role (audited as "ops-team")
user promote 8f47cdb6-9f7e-214d-3fc7-83cfefaff433 admin --actor ops-team

# Remove a role
user demote 8f47cdb6-9f7e-214d-3fc7-83cfefaff433 admin

# List all roles as JSON
user roles list --output json

# Show the role inheritance tree
user roles hierarchy
```

## Audit log

Every `promote` and `demote` is written to `rbac_audit_log` with the timestamp,
user ID, role, actor, and success status. Use `--actor` to identify who performed
the action (username, team name, script name, etc.).
