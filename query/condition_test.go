package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func getTestDialects() map[string]database.Dialect {
	return map[string]database.Dialect{
		"postgres": &postgres.PostgresDialect{},
		"mysql":    &mysql.MySQLDialect{},
		"sqlite":   &sqlite.SQLiteDialect{},
	}
}

func TestEqCondition(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		expected string
	}{
		{"postgres", "postgres", `"age" = $1`},
		{"mysql", "mysql", "`age` = ?"},
		{"sqlite", "sqlite", `"age" = ?`},
	}

	dialects := getTestDialects()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := Eq("age", 25)
			sql, args, nextParam := cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 1 || args[0] != 25 {
				t.Errorf("Expected args [25], got %v", args)
			}

			if nextParam != 2 {
				t.Errorf("Expected nextParam 2, got %d", nextParam)
			}
		})
	}
}

func TestNeCondition(t *testing.T) {
	dialects := getTestDialects()
	cond := Ne("status", "inactive")

	sql, args, nextParam := cond.ToSQL(dialects["postgres"], 1)
	if sql != `"status" != $1` {
		t.Errorf("Expected SQL %q, got %q", `"status" != $1`, sql)
	}
	if len(args) != 1 || args[0] != "inactive" {
		t.Errorf("Expected args [inactive], got %v", args)
	}
	if nextParam != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam)
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		cond     Condition
		expected string
	}{
		{"Gt", Gt("price", 100), `"price" > $1`},
		{"Gte", Gte("price", 100), `"price" >= $1`},
		{"Lt", Lt("price", 100), `"price" < $1`},
		{"Lte", Lte("price", 100), `"price" <= $1`},
	}

	dialect := &postgres.PostgresDialect{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, nextParam := tt.cond.ToSQL(dialect, 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 1 || args[0] != 100 {
				t.Errorf("Expected args [100], got %v", args)
			}

			if nextParam != 2 {
				t.Errorf("Expected nextParam 2, got %d", nextParam)
			}
		})
	}
}

func TestLikeCondition(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		dialect  string
		expected string
	}{
		{"postgres", "postgres", `"name" LIKE $1`},
		{"mysql", "mysql", "`name` LIKE ?"},
		{"sqlite", "sqlite", `"name" LIKE ?`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := Like("name", "John%")
			sql, args, nextParam := cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 1 || args[0] != "John%" {
				t.Errorf("Expected args [John%%], got %v", args)
			}

			if nextParam != 2 {
				t.Errorf("Expected nextParam 2, got %d", nextParam)
			}
		})
	}
}

func TestNotLikeCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := NotLike("email", "%@gmail.com")
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `"email" NOT LIKE $1`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 1 || args[0] != "%@gmail.com" {
		t.Errorf("Expected args [%%@gmail.com], got %v", args)
	}

	if nextParam != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam)
	}
}

func TestILikeCondition(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected string
	}{
		{
			name:     "postgres with ILIKE",
			dialect:  &postgres.PostgresDialect{},
			expected: `"name" ILIKE $1`,
		},
		{
			name:     "mysql with LOWER",
			dialect:  &mysql.MySQLDialect{},
			expected: "LOWER(`name`) LIKE LOWER(?)",
		},
		{
			name:     "sqlite with LOWER",
			dialect:  &sqlite.SQLiteDialect{},
			expected: `LOWER("name") LIKE LOWER(?)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := ILike("name", "john%")
			sql, args, nextParam := cond.ToSQL(tt.dialect, 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 1 || args[0] != "john%" {
				t.Errorf("Expected args [john%%], got %v", args)
			}

			if nextParam != 2 {
				t.Errorf("Expected nextParam 2, got %d", nextParam)
			}
		})
	}
}

func TestNullConditions(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		cond     Condition
		dialect  string
		expected string
	}{
		{"IsNull postgres", IsNull("deleted_at"), "postgres", `"deleted_at" IS NULL`},
		{"IsNull mysql", IsNull("deleted_at"), "mysql", "`deleted_at` IS NULL"},
		{"IsNotNull postgres", IsNotNull("created_at"), "postgres", `"created_at" IS NOT NULL`},
		{"IsNotNull sqlite", IsNotNull("created_at"), "sqlite", `"created_at" IS NOT NULL`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, nextParam := tt.cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if args != nil {
				t.Errorf("Expected nil args, got %v", args)
			}

			if nextParam != 1 {
				t.Errorf("Expected nextParam 1, got %d", nextParam)
			}
		})
	}
}

func TestInCondition(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		dialect  string
		expected string
	}{
		{"postgres", "postgres", `"status" IN ($1, $2, $3)`},
		{"mysql", "mysql", "`status` IN (?, ?, ?)"},
		{"sqlite", "sqlite", `"status" IN (?, ?, ?)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := In("status", "active", "pending", "completed")
			sql, args, nextParam := cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 3 {
				t.Errorf("Expected 3 args, got %d", len(args))
			}

			if nextParam != 4 {
				t.Errorf("Expected nextParam 4, got %d", nextParam)
			}
		})
	}
}

func TestInConditionEmpty(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := In("status")
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	if sql != "1=0" {
		t.Errorf("Expected SQL '1=0' for empty IN, got %q", sql)
	}

	if args != nil {
		t.Errorf("Expected nil args for empty IN, got %v", args)
	}

	if nextParam != 1 {
		t.Errorf("Expected nextParam 1, got %d", nextParam)
	}
}

func TestNotInCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := NotIn("id", 1, 2, 3)
	sql, args, nextParam := cond.ToSQL(dialect, 5)

	expected := `"id" NOT IN ($5, $6, $7)`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if nextParam != 8 {
		t.Errorf("Expected nextParam 8, got %d", nextParam)
	}
}

func TestNotInConditionEmpty(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := NotIn("status")
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	if sql != "1=1" {
		t.Errorf("Expected SQL '1=1' for empty NOT IN, got %q", sql)
	}

	if args != nil {
		t.Errorf("Expected nil args for empty NOT IN, got %v", args)
	}

	if nextParam != 1 {
		t.Errorf("Expected nextParam 1, got %d", nextParam)
	}
}

func TestBetweenCondition(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		dialect  string
		expected string
	}{
		{"postgres", "postgres", `"created_at" BETWEEN $1 AND $2`},
		{"mysql", "mysql", "`created_at` BETWEEN ? AND ?"},
		{"sqlite", "sqlite", `"created_at" BETWEEN ? AND ?`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := Between("created_at", "2024-01-01", "2024-12-31")
			sql, args, nextParam := cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 2 {
				t.Errorf("Expected 2 args, got %d", len(args))
			}

			if nextParam != 3 {
				t.Errorf("Expected nextParam 3, got %d", nextParam)
			}
		})
	}
}

func TestAndCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	cond := And(
		Eq("status", "active"),
		Gt("age", 18),
		Like("name", "John%"),
	)

	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `("status" = $1 AND "age" > $2 AND "name" LIKE $3)`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if nextParam != 4 {
		t.Errorf("Expected nextParam 4, got %d", nextParam)
	}
}

func TestOrCondition(t *testing.T) {
	dialect := &mysql.MySQLDialect{}

	cond := Or(
		Eq("type", "admin"),
		Eq("type", "moderator"),
	)

	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := "(`type` = ? OR `type` = ?)"
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if nextParam != 3 {
		t.Errorf("Expected nextParam 3, got %d", nextParam)
	}
}

func TestNestedAndOrConditions(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	cond := And(
		Eq("status", "active"),
		Or(
			Eq("role", "admin"),
			Eq("role", "moderator"),
		),
		Gt("age", 18),
	)

	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `("status" = $1 AND ("role" = $2 OR "role" = $3) AND "age" > $4)`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d", len(args))
	}

	if nextParam != 5 {
		t.Errorf("Expected nextParam 5, got %d", nextParam)
	}
}

func TestEmptyAndCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := And()
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	if sql != "1=1" {
		t.Errorf("Expected SQL '1=1' for empty AND, got %q", sql)
	}

	if args != nil {
		t.Errorf("Expected nil args, got %v", args)
	}

	if nextParam != 1 {
		t.Errorf("Expected nextParam 1, got %d", nextParam)
	}
}

func TestSingleConditionInAnd(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := And(Eq("status", "active"))
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `"status" = $1`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}

	if nextParam != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam)
	}
}

func TestNotCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := Not(Eq("status", "deleted"))
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `NOT ("status" = $1)`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}

	if nextParam != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam)
	}
}

func TestNotWithComplexCondition(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := Not(And(
		Eq("status", "deleted"),
		IsNull("deleted_at"),
	))
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `NOT (("status" = $1 AND "deleted_at" IS NULL))`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}

	if nextParam != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam)
	}
}

func TestRawCondition(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		dialect  string
		expected string
	}{
		{"postgres", "postgres", "age > $1 AND status = $2"},
		{"mysql", "mysql", "age > ? AND status = ?"},
		{"sqlite", "sqlite", "age > ? AND status = ?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := Raw("age > ? AND status = ?", 18, "active")
			sql, args, nextParam := cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if len(args) != 2 {
				t.Errorf("Expected 2 args, got %d", len(args))
			}

			if nextParam != 3 {
				t.Errorf("Expected nextParam 3, got %d", nextParam)
			}
		})
	}
}

func TestRawConditionWithOffset(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := Raw("price BETWEEN ? AND ?", 100, 500)
	sql, args, nextParam := cond.ToSQL(dialect, 5)

	expected := "price BETWEEN $5 AND $6"
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if nextParam != 7 {
		t.Errorf("Expected nextParam 7, got %d", nextParam)
	}
}

func TestRawConditionNoPlaceholders(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	cond := Raw("1=1")
	sql, args, nextParam := cond.ToSQL(dialect, 1)

	if sql != "1=1" {
		t.Errorf("Expected SQL '1=1', got %q", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}

	if nextParam != 1 {
		t.Errorf("Expected nextParam 1, got %d", nextParam)
	}
}

func TestColumnComparisonConditions(t *testing.T) {
	dialects := getTestDialects()

	tests := []struct {
		name     string
		cond     Condition
		dialect  string
		expected string
	}{
		{"ColEq postgres", ColEq("created_at", "updated_at"), "postgres", `"created_at" = "updated_at"`},
		{"ColEq mysql", ColEq("created_at", "updated_at"), "mysql", "`created_at` = `updated_at`"},
		{"ColNe postgres", ColNe("start_date", "end_date"), "postgres", `"start_date" != "end_date"`},
		{"ColGt sqlite", ColGt("price", "discount"), "sqlite", `"price" > "discount"`},
		{"ColLt mysql", ColLt("min_value", "max_value"), "mysql", "`min_value` < `max_value`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, nextParam := tt.cond.ToSQL(dialects[tt.dialect], 1)

			if sql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, sql)
			}

			if args != nil {
				t.Errorf("Expected nil args, got %v", args)
			}

			if nextParam != 1 {
				t.Errorf("Expected nextParam 1, got %d", nextParam)
			}
		})
	}
}

func TestParameterCounting(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	cond1 := Eq("status", "active")
	sql1, args1, nextParam1 := cond1.ToSQL(dialect, 1)

	if sql1 != `"status" = $1` {
		t.Errorf("Expected SQL %q, got %q", `"status" = $1`, sql1)
	}
	if nextParam1 != 2 {
		t.Errorf("Expected nextParam 2, got %d", nextParam1)
	}

	cond2 := Gt("age", 18)
	sql2, args2, nextParam2 := cond2.ToSQL(dialect, nextParam1)

	if sql2 != `"age" > $2` {
		t.Errorf("Expected SQL %q, got %q", `"age" > $2`, sql2)
	}
	if nextParam2 != 3 {
		t.Errorf("Expected nextParam 3, got %d", nextParam2)
	}

	allArgs := append(args1, args2...)
	if len(allArgs) != 2 {
		t.Errorf("Expected 2 total args, got %d", len(allArgs))
	}
}

func TestComplexNestedConditions(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	cond := Or(
		And(
			Eq("type", "user"),
			Gte("age", 18),
			IsNotNull("email"),
		),
		And(
			Eq("type", "admin"),
			IsNull("deleted_at"),
		),
	)

	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `(("type" = $1 AND "age" >= $2 AND "email" IS NOT NULL) OR ("type" = $3 AND "deleted_at" IS NULL))`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	expectedArgs := []any{"user", 18, "admin"}
	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Expected arg %d to be %v, got %v", i, expectedArgs[i], arg)
		}
	}

	if nextParam != 4 {
		t.Errorf("Expected nextParam 4, got %d", nextParam)
	}
}

func TestMixedConditionsWithNilValues(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	cond := And(
		Eq("name", "John"),
		IsNull("deleted_at"),
		In("status", "active", "pending"),
		ColEq("created_by", "updated_by"),
	)

	sql, args, nextParam := cond.ToSQL(dialect, 1)

	expected := `("name" = $1 AND "deleted_at" IS NULL AND "status" IN ($2, $3) AND "created_by" = "updated_by")`
	if sql != expected {
		t.Errorf("Expected SQL %q, got %q", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if nextParam != 4 {
		t.Errorf("Expected nextParam 4, got %d", nextParam)
	}
}

func TestAllDialectsConsistency(t *testing.T) {
	dialects := getTestDialects()
	cond := And(
		Eq("status", "active"),
		Gt("age", 18),
		In("role", "admin", "moderator"),
		IsNotNull("email"),
	)

	for name, dialect := range dialects {
		t.Run(name, func(t *testing.T) {
			sql, args, nextParam := cond.ToSQL(dialect, 1)

			if sql == "" {
				t.Errorf("SQL should not be empty for dialect %s", name)
			}

			if len(args) != 4 {
				t.Errorf("Expected 4 args for dialect %s, got %d", name, len(args))
			}

			if nextParam != 5 {
				t.Errorf("Expected nextParam 5 for dialect %s, got %d", name, nextParam)
			}
		})
	}
}
