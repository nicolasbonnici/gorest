# Security

## Reporting a vulnerability

Report suspected vulnerabilities privately rather than opening a public issue.
Include a reproduction and the affected version; a fix or mitigation will be
acknowledged before any public disclosure.

## Running the checks

Every repository in the GoREST workspace carries the same three static gates,
locally and in CI:

```bash
make security          # all three
make security-sast     # gosec, via golangci-lint
make security-vuln     # govulncheck, including stdlib CVEs
make security-secrets  # gitleaks
```

`.github/workflows/security.yml` runs them on every pull request and push, and
weekly on a schedule: govulncheck compares the code against a database that
moves on its own, so a repository with no commits can still become vulnerable
between runs.

Scanner versions are pinned in each `Makefile` and workflow so a local run and a
CI run judge the same code the same way.

`.gitleaks.toml` allowlists documentation placeholders and the throwaway
credentials for local test containers. It does **not** exempt whole files or
directories: a real key pasted into a README is still caught.

The `blog` reference application adds a pentest suite (`make pentest`) that
drives a running API over HTTP. See its `CLAUDE.md`.

## What the framework enforces

### Configuration

`config.Validate()` refuses to start a **production** server that is configured
like a development one:

| Setting | Requirement in production |
|---------|---------------------------|
| `server.cors_origins` | must not be `*` |
| `server.ratelimit_enabled` | must be `true` |
| `server.scheme` | must be `https` |
| `auth.jwt_secret` | at least 32 characters, and not a recognisable placeholder |

These are checked only when `server.environment` is `production`, so a local run
is unaffected.

### Request handling

- A **recover** middleware wraps the whole chain. A panic in any handler fails
  that one request instead of taking the process down.
- Security headers (`X-Frame-Options`, `X-Content-Type-Options`, `CSP`, `HSTS`,
  `Referrer-Policy`, `Permissions-Policy`) are set by `middleware.Security()`.
- `X-Powered-By` carries the framework version only in development. Elsewhere
  `gorest.Start` suppresses it, since a version string mainly helps someone
  match a deployment against a CVE list.
- Two rate limiters, because they catch different things: the server-wide one
  counts requests per second and stops floods, and
  `auth/middleware.CredentialRateLimit` counts a handful of attempts over
  minutes on `/auth/register`, `/auth/login` and `/auth/refresh`, which is what
  password guessing actually looks like.

### Authentication

- Access tokens are HS256 JWTs, validated with an explicit algorithm allowlist.
  `alg: none`, a stripped signature, and RS256-to-HS256 confusion are all
  rejected before any signature work. See `auth/jwt/attacks_test.go`.
- Login answers an unknown account and a wrong password identically, and runs a
  bcrypt comparison on both paths. Returning faster for an unknown address
  enumerates accounts just as well as a distinct status code does.
- Refresh tokens are opaque, stored as a SHA-256 hash, rotated on every use,
  and a replayed token revokes the whole family.

### Authorization

`rbac.RoleLoader`, `rbac.RequireAuthenticated`, `rbac.RequireRole` and
`rbac.RequireAnyRole` are the shared route guards. They live in the core
precisely because they used to be copied into each plugin, and two plugins
ended up registering their mutating routes with no guard at all.

**A plugin that exposes mutating routes must mount them.** Reads may be public;
writes must not be.

### SQL

All queries go through `query.New(dialect)`, which parameterises every value.
Filter, sort and expand parameters are parsed against a per-resource allowlist,
and `filter/fuzz_test.go` fuzzes that parser with the invariants that matter:
no panic, no unparameterised value in the generated clause, and nothing outside
the allowlist reaching a column name.

```bash
go test ./filter/ -run=NONE -fuzz=FuzzFilterParsing -fuzztime=60s
```

## Fixed in v0.7

| Issue | Where |
|-------|-------|
| Unauthenticated remote DoS: a malformed filter key (`?][=1`) panicked the parser and, with no recover middleware, killed the process | `filter/filters.go`, `gorest.go` |
| Account enumeration: an unknown address returned 500 where a wrong password returned 401, and skipped bcrypt so it also answered faster | `auth/handlers/auth.go` |
| Mutating routes registered with no authorization guard, so anyone could upload, edit or delete anonymously | `gorest-media`, `gorest-taxonomy` |
| Uploads defaulted to accepting every MIME type, including scripts and scriptable SVG | `gorest-media` |
| Path traversal through the `ENVIRONMENT` variable into an arbitrary YAML file | `config/loader.go` |
| Credential endpoints had no throttle distinct from the global flood limiter | `auth/middleware/ratelimit.go` |
| `X-Powered-By` advertised the exact framework version on every response | `response/response.go` |
| Applications built without a pinned toolchain shipped a stdlib carrying 10 known CVEs | `gorest-mcp`, `blog` |
