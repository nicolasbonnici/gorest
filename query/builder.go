// Package query provides a fluent, type-safe SQL query builder for GoREST.
//
// This package implements a builder pattern for constructing SQL queries
// in a database-agnostic way. It supports multiple SQL dialects (PostgreSQL,
// MySQL, SQLite) and provides specialized builders for different SQL operations.
//
// Usage:
//
//	dialect := &postgres.PostgresDialect{}
//	builder := query.New(dialect)
//
//	// Build a SELECT query
//	selectQuery := builder.Select("id", "name", "email").From("users")
//
//	// Build an INSERT query
//	insertQuery := builder.Insert("users").Values(...)
//
//	// Build an UPDATE query
//	updateQuery := builder.Update("users").Set("name", "John")
//
//	// Build a DELETE query
//	deleteQuery := builder.Delete("users").Where("id = ?", 1)
package query

import "github.com/nicolasbonnici/gorest/database"

type Builder struct {
	dialect database.Dialect
}

func New(dialect database.Dialect) *Builder {
	return &Builder{dialect: dialect}
}

func (b *Builder) Select(columns ...string) *SelectBuilder {
	sb := &SelectBuilder{
		dialect:          b.dialect,
		columns:          columns,
		columnExprs:      make([]Expression, 0),
		joins:            make([]joinClause, 0),
		conditions:       make([]Condition, 0),
		groupBy:          make([]string, 0),
		groupByExprs:     make([]Expression, 0),
		havingConditions: make([]Condition, 0),
		orderBy:          make([]orderClause, 0),
		orderByExprs:     make([]orderByExpr, 0),
	}

	for _, col := range columns {
		if sb.err == nil {
			sb.err = ValidateIdentifier(col)
		}
	}

	return sb
}

func (b *Builder) Insert(table string) *InsertBuilder {
	return &InsertBuilder{
		dialect:   b.dialect,
		table:     table,
		columns:   make([]string, 0),
		values:    make([][]any, 0),
		returning: make([]string, 0),
	}
}

func (b *Builder) Update(table string) *UpdateBuilder {
	return &UpdateBuilder{
		dialect:    b.dialect,
		table:      table,
		sets:       make(map[string]any),
		setOrder:   make([]string, 0),
		conditions: make([]Condition, 0),
		returning:  make([]string, 0),
	}
}

func (b *Builder) Delete(table string) *DeleteBuilder {
	return &DeleteBuilder{
		dialect:    b.dialect,
		table:      table,
		conditions: make([]Condition, 0),
		returning:  make([]string, 0),
	}
}

type Order int

const (
	ASC Order = iota
	DESC
)

func (o Order) String() string {
	switch o {
	case DESC:
		return "DESC"
	default:
		return "ASC"
	}
}

type orderClause struct {
	column    string
	direction Order
}
