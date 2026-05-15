package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/rbac"
)

func main() {
	configDir := "."
	dsn := os.Getenv("DATABASE_URL")
	outputFmt := "table"
	actor := "cli"

	args := os.Args[1:]
	rest := args[:0]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			i++
			if i < len(args) {
				configDir = args[i]
			}
		case "--dsn":
			i++
			if i < len(args) {
				dsn = args[i]
			}
		case "--output", "-o":
			i++
			if i < len(args) {
				outputFmt = args[i]
			}
		case "--actor":
			i++
			if i < len(args) {
				actor = args[i]
			}
		default:
			rest = append(rest, args[i])
		}
	}

	if len(rest) == 0 || rest[0] == "help" || rest[0] == "--help" || rest[0] == "-h" {
		printUsage()
		return
	}

	// Connect
	if dsn == "" {
		cfg, err := config.Load(configDir)
		if err != nil {
			fatalf("loading config from %q: %v", configDir, err)
		}
		dsn = cfg.Database.URL
	}
	if dsn == "" {
		fatalf("no database URL — use --dsn, set DATABASE_URL, or point --config to a gorest.yaml directory")
	}

	db, err := database.Open("", dsn)
	if err != nil {
		fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	repo := rbac.NewRepository(db)
	ctx := context.Background()

	if err := dispatch(ctx, repo, rest, outputFmt, actor); err != nil {
		fatalf("%v", err)
	}
}

func dispatch(ctx context.Context, repo *rbac.Repository, args []string, format, actor string) error {
	switch args[0] {
	case "list":
		return cmdUserList(ctx, repo, format)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: user show <user-id>")
		}
		return cmdUserShow(ctx, repo, args[1], format)
	case "promote":
		if len(args) < 3 {
			return fmt.Errorf("usage: user promote <user-id> <role>")
		}
		return cmdUserPromote(ctx, repo, args[1], args[2], actor)
	case "demote":
		if len(args) < 3 {
			return fmt.Errorf("usage: user demote <user-id> <role>")
		}
		return cmdUserDemote(ctx, repo, args[1], args[2], actor)
	case "roles":
		if len(args) < 2 {
			return fmt.Errorf("usage: user roles <list|hierarchy>")
		}
		switch args[1] {
		case "list":
			return cmdRolesList(ctx, repo, format)
		case "hierarchy":
			return cmdRolesHierarchy(ctx, repo, format)
		default:
			return fmt.Errorf("unknown roles subcommand %q — use 'list' or 'hierarchy'", args[1])
		}
	default:
		return fmt.Errorf("unknown command %q — run 'user help' for usage", args[0])
	}
}

// ── user list ────────────────────────────────────────────────────────────────

func cmdUserList(ctx context.Context, repo *rbac.Repository, format string) error {
	users, err := repo.ListUsers(ctx)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		fmt.Println("No users found.")
		return nil
	}

	if format == "json" {
		return printJSON(users)
	}

	printTable(
		[]string{"USER ID", "EMAIL", "ROLES", "LAST ASSIGNED"},
		func(row func(...string)) {
			for _, u := range users {
				roles := strings.Join(u.Roles, ", ")
				if roles == "" {
					roles = "(none)"
				}
				updated := ""
				if !u.UpdatedAt.IsZero() {
					updated = u.UpdatedAt.Format(time.DateTime)
				}
				row(u.UserID, u.Email, roles, updated)
			}
		},
	)
	return nil
}

// ── user show ────────────────────────────────────────────────────────────────

func cmdUserShow(ctx context.Context, repo *rbac.Repository, userID, format string) error {
	roles, err := repo.GetUserRoles(ctx, userID)
	if err != nil {
		return err
	}

	type result struct {
		UserID string   `json:"user_id"`
		Roles  []string `json:"roles"`
	}
	data := result{UserID: userID, Roles: roles}

	if format == "json" {
		return printJSON(data)
	}

	if len(roles) == 0 {
		fmt.Printf("User %s has no roles.\n", userID)
		return nil
	}

	fmt.Printf("User:  %s\n", userID)
	fmt.Printf("Roles: %s\n", strings.Join(roles, ", "))
	return nil
}

// ── user promote / demote ─────────────────────────────────────────────────────

func cmdUserPromote(ctx context.Context, repo *rbac.Repository, userID, role, actor string) error {
	if err := repo.AssignRole(ctx, userID, role, actor); err != nil {
		return err
	}
	fmt.Printf("Assigned role %q to user %s.\n", role, userID)
	return nil
}

func cmdUserDemote(ctx context.Context, repo *rbac.Repository, userID, role, actor string) error {
	if err := repo.RemoveRole(ctx, userID, role, actor); err != nil {
		return err
	}
	fmt.Printf("Removed role %q from user %s.\n", role, userID)
	return nil
}

// ── user roles list ───────────────────────────────────────────────────────────

func cmdRolesList(ctx context.Context, repo *rbac.Repository, format string) error {
	roles, err := repo.ListRoles(ctx)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		fmt.Println("No roles defined.")
		return nil
	}

	if format == "json" {
		return printJSON(roles)
	}

	printTable(
		[]string{"NAME", "DESCRIPTION", "PARENT"},
		func(row func(...string)) {
			for _, r := range roles {
				parent := r.Parent
				if parent == "" {
					parent = "(root)"
				}
				row(r.Name, r.Description, parent)
			}
		},
	)
	return nil
}

// ── user roles hierarchy ──────────────────────────────────────────────────────

func cmdRolesHierarchy(ctx context.Context, repo *rbac.Repository, format string) error {
	hierarchy, err := repo.GetRoleHierarchy(ctx)
	if err != nil {
		return err
	}

	if format == "json" {
		return printJSON(hierarchy)
	}

	if len(hierarchy) == 0 {
		fmt.Println("No role hierarchy defined.")
		return nil
	}

	// Find roots: parents that are not children of any other role.
	allChildren := map[string]bool{}
	for _, children := range hierarchy {
		for _, c := range children {
			allChildren[c] = true
		}
	}

	visited := map[string]bool{}
	for parent := range hierarchy {
		if !allChildren[parent] {
			printTree(parent, hierarchy, "", visited)
		}
	}
	// Catch any remaining unvisited nodes (orphaned subtrees).
	for parent := range hierarchy {
		if !visited[parent] {
			printTree(parent, hierarchy, "", visited)
		}
	}
	return nil
}

func printTree(role string, hierarchy map[string][]string, prefix string, visited map[string]bool) {
	if visited[role] {
		return
	}
	visited[role] = true
	fmt.Printf("%s%s\n", prefix, role)

	children := hierarchy[role]
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}
		fmt.Printf("%s%s", prefix, connector)
		printTree(child, hierarchy, childPrefix, visited)
	}
}

// ── output helpers ────────────────────────────────────────────────────────────

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printTable(headers []string, fill func(row func(...string))) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	// Two-pass: first measure, then print.
	var rows [][]string
	fill(func(cols ...string) {
		for i, c := range cols {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
		rows = append(rows, cols)
	})

	fmtRow := func(cols []string) {
		for i, w := range widths {
			val := ""
			if i < len(cols) {
				val = cols[i]
			}
			if i < len(widths)-1 {
				fmt.Printf("%-*s  ", w, val)
			} else {
				fmt.Printf("%s\n", val)
			}
		}
	}

	fmtRow(headers)
	sep := make([]string, len(headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	fmtRow(sep)

	for _, r := range rows {
		fmtRow(r)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`GoREST User & Role Management CLI

Usage:
  user <command> [arguments]

User commands:
  user list                        List all users and their roles
  user show <user-id>              Show roles for a specific user
  user promote <user-id> <role>    Assign a role to a user
  user demote  <user-id> <role>    Remove a role from a user

Role commands:
  user roles list                  List all defined roles
  user roles hierarchy             Display the role inheritance tree

Global flags:
  --config <dir>    Directory containing gorest.yaml (default: .)
  --dsn <url>       Raw database URL (overrides config)
  --output, -o      Output format: table (default) or json
  --actor <name>    Identity recorded in audit log for promote/demote (default: cli)

Examples:
  user list --config /var/apps/myapi
  user show 8f47cdb6-9f7e-214d-3fc7-83cfefaff433
  user promote 8f47cdb6-9f7e-214d-3fc7-83cfefaff433 admin --actor ops-team
  user demote  8f47cdb6-9f7e-214d-3fc7-83cfefaff433 admin
  user roles hierarchy --output json
`)
}
