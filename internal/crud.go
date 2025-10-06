package crud

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/nicolasbonnici/gorest/internal/models"
)

type CRUD[T models.Model] struct {
	DB *sql.DB
}

func New[T models.Model](db *sql.DB) *CRUD[T] {
	return &CRUD[T]{DB: db}
}

func (c *CRUD[T]) Create(ctx context.Context, m T) error {
	v := reflect.ValueOf(m)
	t := reflect.TypeOf(m)

	var cols []string
	var vals []interface{}
	var placeholders []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "id" {
			continue
		}
		cols = append(cols, tag)
		vals = append(vals, v.Field(i).Interface())
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(vals)))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		m.TableName(),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
	_, err := c.DB.ExecContext(ctx, query, vals...)
	return err
}


type Repository[T any] interface {
	Create(ctx context.Context, m T) error
	GetAll(ctx context.Context) ([]T, error)
	GetByID(ctx context.Context, id any) (*T, error)
	Update(ctx context.Context, id any, m T) error
	Delete(ctx context.Context, id any) error
}