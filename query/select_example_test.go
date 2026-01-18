package query_test

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/query"
)

func ExampleSelectBuilder_simple() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select("id", "name", "email").
		From("users").
		Build()

	fmt.Println(sql)
	fmt.Println(len(args))

	// Output:
	// SELECT "id", "name", "email" FROM "users"
	// 0
}

func ExampleSelectBuilder_where() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select().
		From("users").
		Where(query.Eq("status", "active")).
		And(query.Gt("age", 18)).
		Build()

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// SELECT * FROM "users" WHERE "status" = $1 AND "age" > $2
	// [active 18]
}

func ExampleSelectBuilder_orderByLimit() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select("id", "name", "created_at").
		From("users").
		OrderBy("created_at", query.DESC).
		Limit(10).
		Build()

	fmt.Println(sql)
	fmt.Println(len(args))

	// Output:
	// SELECT "id", "name", "created_at" FROM "users" ORDER BY "created_at" DESC LIMIT 10
	// 0
}

func ExampleSelectBuilder_distinct() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select("country").
		Distinct().
		From("users").
		OrderBy("country", query.ASC).
		Build()

	fmt.Println(sql)
	fmt.Println(len(args))

	// Output:
	// SELECT DISTINCT "country" FROM "users" ORDER BY "country" ASC
	// 0
}

func ExampleSelectBuilder_alias() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select("u.id", "u.name").
		From("users").
		As("u").
		Build()

	fmt.Println(sql)
	fmt.Println(len(args))

	// Output:
	// SELECT "u"."id", "u"."name" FROM "users" AS "u"
	// 0
}

func ExampleSelectBuilder_complex() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select("id", "name", "email", "created_at").
		From("users").
		As("u").
		Where(query.Eq("status", "active")).
		And(query.In("role", "admin", "moderator")).
		And(query.Between("age", 18, 65)).
		OrderBy("created_at", query.DESC).
		Limit(10).
		Offset(20).
		Build()

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// SELECT "id", "name", "email", "created_at" FROM "users" AS "u" WHERE "status" = $1 AND "role" IN ($2, $3) AND "age" BETWEEN $4 AND $5 ORDER BY "created_at" DESC LIMIT 10 OFFSET 20
	// [active admin moderator 18 65]
}

func ExampleSelectBuilder_or() {
	builder := query.New(&postgres.PostgresDialect{})

	sql, args, _ := builder.Select().
		From("users").
		Where(query.Eq("status", "active")).
		Or(query.Eq("status", "pending")).
		Build()

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// SELECT * FROM "users" WHERE ("status" = $1 OR "status" = $2)
	// [active pending]
}
