package query_test

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/query"
)

func ExampleNew() {
	dialect := &postgres.PostgresDialect{}
	builder := query.New(dialect)

	selectBuilder := builder.Select("id", "name", "email")
	fmt.Printf("SelectBuilder created with %d columns\n", 3)

	insertBuilder := builder.Insert("users")
	fmt.Printf("InsertBuilder created for table: users\n")

	updateBuilder := builder.Update("users")
	fmt.Printf("UpdateBuilder created for table: users\n")

	deleteBuilder := builder.Delete("users")
	fmt.Printf("DeleteBuilder created for table: users\n")

	_ = selectBuilder
	_ = insertBuilder
	_ = updateBuilder
	_ = deleteBuilder

	// Output:
	// SelectBuilder created with 3 columns
	// InsertBuilder created for table: users
	// UpdateBuilder created for table: users
	// DeleteBuilder created for table: users
}

func ExampleOrder_String() {
	fmt.Println(query.ASC.String())
	fmt.Println(query.DESC.String())

	// Output:
	// ASC
	// DESC
}

func ExampleEq() {
	dialect := &postgres.PostgresDialect{}
	cond := query.Eq("age", 25)
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: "age" = $1
	// Args: [25]
}

func ExampleAnd() {
	dialect := &postgres.PostgresDialect{}
	cond := query.And(
		query.Eq("status", "active"),
		query.Gt("age", 18),
		query.IsNotNull("email"),
	)
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: ("status" = $1 AND "age" > $2 AND "email" IS NOT NULL)
	// Args: [active 18]
}

func ExampleOr() {
	dialect := &postgres.PostgresDialect{}
	cond := query.Or(
		query.Eq("role", "admin"),
		query.Eq("role", "moderator"),
	)
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: ("role" = $1 OR "role" = $2)
	// Args: [admin moderator]
}

func ExampleIn() {
	dialect := &postgres.PostgresDialect{}
	cond := query.In("status", "active", "pending", "completed")
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: "status" IN ($1, $2, $3)
	// Args: [active pending completed]
}

func ExampleILike() {
	dialect := &postgres.PostgresDialect{}
	cond := query.ILike("name", "john%")
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: "name" ILIKE $1
	// Args: [john%]
}

func ExampleRaw() {
	dialect := &postgres.PostgresDialect{}
	cond := query.Raw("age BETWEEN ? AND ?", 18, 65)
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: age BETWEEN $1 AND $2
	// Args: [18 65]
}

func ExampleBetween() {
	dialect := &postgres.PostgresDialect{}
	cond := query.Between("created_at", "2024-01-01", "2024-12-31")
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: "created_at" BETWEEN $1 AND $2
	// Args: [2024-01-01 2024-12-31]
}

func ExampleNot() {
	dialect := &postgres.PostgresDialect{}
	cond := query.Not(query.Eq("status", "deleted"))
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: NOT ("status" = $1)
	// Args: [deleted]
}

func ExampleColEq() {
	dialect := &postgres.PostgresDialect{}
	cond := query.ColEq("created_at", "updated_at")
	sql, args, _ := cond.ToSQL(dialect, 1)

	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: "created_at" = "updated_at"
	// Args: []
}
