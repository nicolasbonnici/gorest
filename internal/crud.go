package internal

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/gen/models"
)

type CRUD[T models.Model] struct {
	DB *pgxpool.Pool
}

func New[T models.Model](db *pgxpool.Pool) *CRUD[T] {
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
		if tag == "" || tag == "id" || tag == "created_at" || tag == "updated_at" {
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
	_, err := c.DB.Exec(ctx, query, vals...)
	return err
}

func (c *CRUD[T]) GetAll(ctx context.Context) ([]T, error) {
	var zero T
	t := reflect.TypeOf(zero)

	var cols []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			cols = append(cols, tag)
		}
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), zero.TableName())

	rows, err := c.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		v := reflect.ValueOf(&item).Elem()

		fields := make([]interface{}, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			fields[i] = v.Field(i).Addr().Interface()
		}

		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (c *CRUD[T]) GetByID(ctx context.Context, id any) (*T, error) {
	var item T
	t := reflect.TypeOf(item)

	var cols []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			cols = append(cols, tag)
		}
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", strings.Join(cols, ", "), item.TableName())

	row := c.DB.QueryRow(ctx, query, id)

	v := reflect.ValueOf(&item).Elem()

	fields := make([]interface{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		fields[i] = v.Field(i).Addr().Interface()
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *CRUD[T]) Update(ctx context.Context, id any, m T) error {
	v := reflect.ValueOf(m)
	t := reflect.TypeOf(m)

	var setClauses []string
	var vals []interface{}
	paramIdx := 1

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "id" || tag == "created_at" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", tag, paramIdx))
		vals = append(vals, v.Field(i).Interface())
		paramIdx++
	}

	vals = append(vals, id)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		m.TableName(),
		strings.Join(setClauses, ", "),
		paramIdx,
	)

	_, err := c.DB.Exec(ctx, query, vals...)
	return err
}

func (c *CRUD[T]) Delete(ctx context.Context, id any) error {
	var zero T
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", zero.TableName())
	_, err := c.DB.Exec(ctx, query, id)
	return err
}


type Repository[T any] interface {
	Create(ctx context.Context, m T) error
	GetAll(ctx context.Context) ([]T, error)
	GetByID(ctx context.Context, id any) (*T, error)
	Update(ctx context.Context, id any, m T) error
	Delete(ctx context.Context, id any) error
}