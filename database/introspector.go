package database

import "context"

type SchemaIntrospector interface {
	LoadSchema(ctx context.Context) ([]TableSchema, error)
	GetColumns(ctx context.Context, tableName string) ([]Column, error)
	GetRelations(ctx context.Context) ([]Relation, error)
}

type TableSchema struct {
	TableName string
	Columns   []Column
	Relations []Relation
}

type Column struct {
	Name       string
	Type       string
	IsNullable bool
}

type Relation struct {
	ChildTable   string
	ChildColumn  string
	ParentTable  string
	ParentColumn string
}
