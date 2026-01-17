package main

import (
	"fmt"
	"github.com/nicolasbonnici/gorest/database/dialects/postgres"
	"github.com/nicolasbonnici/gorest/query"
)

func main() {
	dialect := &postgres.PostgresDialect{}
	builder := query.New(dialect)

	// Create a deliberately invalid subquery (SELECT without FROM)
	invalidSubquery := builder.Select("id")
	
	// Before our fix, this would panic. Now it should return an errorCondition
	condition := query.InSubquery("user_id", invalidSubquery)
	
	// Try to use it in a query
	mainQuery := builder.Select("name").From("users").Where(condition)
	
	sql, args, err := mainQuery.Build()
	
	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)
	fmt.Printf("Error: %v\n", err)
	
	// The query should build (not panic) but will contain an error message in the SQL
	if err == nil {
		fmt.Println("\nSUCCESS: No panic occurred! Error was handled gracefully.")
	}
}
