package crud

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal/hooks"
)

type CRUD[T Model] struct {
	DB    *pgxpool.Pool
	Hooks hooks.Hooks[T]
}

func New[T Model](db *pgxpool.Pool) *CRUD[T] {
	return &CRUD[T]{
		DB:    db,
		Hooks: hooks.NoOpHooks[T]{},
	}
}

func NewWithHooks[T Model](db *pgxpool.Pool, h hooks.Hooks[T]) *CRUD[T] {
	return &CRUD[T]{
		DB:    db,
		Hooks: h,
	}
}

func (c *CRUD[T]) Create(ctx context.Context, m T) error {
	if err := c.Hooks.StateProcessor(ctx, hooks.OperationCreate, nil, &m); err != nil {
		return err
	}

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationCreate, nil, &m)

	var query string
	var vals []interface{}

	if skip && customQuery != "" {
		query = customQuery
		vals = customArgs
	} else {
		v := reflect.ValueOf(m)
		t := reflect.TypeOf(m)

		var cols []string
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

		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
			m.TableName(),
			strings.Join(cols, ", "),
			strings.Join(placeholders, ", "),
		)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationCreate, query, vals)
	if err != nil {
		return err
	}

	var execErr error
	var result any

	row := c.DB.QueryRow(ctx, finalQuery, finalArgs...)
	var createdID any
	scanErr := row.Scan(&createdID)
	if scanErr != nil {
		execErr = scanErr
	} else {
		result = createdID
		v := reflect.ValueOf(&m).Elem()
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag == "id" {
				fieldValue := v.Field(i)
				if fieldValue.CanSet() {
					switch fieldValue.Kind() {
					case reflect.String:
						if s, ok := createdID.(string); ok {
							fieldValue.SetString(s)
						} else {
							log.Printf("warning: failed to cast ID to string, got type %T", createdID)
						}
					case reflect.Int, reflect.Int64:
						if n, ok := createdID.(int64); ok {
							fieldValue.SetInt(n)
						} else if n, ok := createdID.(int); ok {
							fieldValue.SetInt(int64(n))
						} else {
							log.Printf("warning: failed to cast ID to int64, got type %T", createdID)
						}
					default:
						log.Printf("warning: unsupported ID field type: %v", fieldValue.Kind())
					}
				}
				break
			}
		}
	}

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationCreate, finalQuery, finalArgs, result, execErr); err != nil {
		return err
	}

	if execErr != nil {
		return execErr
	}

	if err := c.Hooks.SerializeOne(ctx, hooks.OperationCreate, &m); err != nil {
		return err
	}

	return nil
}

func (c *CRUD[T]) GetAll(ctx context.Context) ([]T, error) {
	var zero T

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationGetAll, nil, nil)

	var query string
	var args []any
	var fieldIndices []int

	if skip && customQuery != "" {
		query = customQuery
		args = customArgs
	} else {
		t := reflect.TypeOf(zero)

		var cols []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag != "" {
				cols = append(cols, tag)
				fieldIndices = append(fieldIndices, i)
			}
		}

		query = fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), zero.TableName())
		args = []any{}
	}

	if skip && customQuery != "" {
		t := reflect.TypeOf(zero)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag != "" {
				fieldIndices = append(fieldIndices, i)
			}
		}
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetAll, query, args)
	if err != nil {
		return nil, err
	}

	rows, execErr := c.DB.Query(ctx, finalQuery, finalArgs...)
	if execErr != nil {
		_ = c.Hooks.AfterQuery(ctx, hooks.OperationGetAll, finalQuery, finalArgs, nil, execErr)
		return nil, execErr
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		v := reflect.ValueOf(&item).Elem()

		fields := make([]interface{}, len(fieldIndices))
		for i, idx := range fieldIndices {
			fields[i] = v.Field(idx).Addr().Interface()
		}

		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationGetAll, finalQuery, finalArgs, results, nil); err != nil {
		return nil, err
	}

	if err := c.Hooks.SerializeMany(ctx, hooks.OperationGetAll, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (c *CRUD[T]) GetByID(ctx context.Context, id any) (*T, error) {
	var item T

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationGetByID, id, nil)

	var query string
	var args []any
	var fieldIndices []int

	if skip && customQuery != "" {
		query = customQuery
		args = customArgs
	} else {
		t := reflect.TypeOf(item)

		var cols []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag != "" {
				cols = append(cols, tag)
				fieldIndices = append(fieldIndices, i)
			}
		}

		query = fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", strings.Join(cols, ", "), item.TableName())
		args = []any{id}
	}

	if skip && customQuery != "" {
		t := reflect.TypeOf(item)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag != "" {
				fieldIndices = append(fieldIndices, i)
			}
		}
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetByID, query, args)
	if err != nil {
		return nil, err
	}

	row := c.DB.QueryRow(ctx, finalQuery, finalArgs...)

	v := reflect.ValueOf(&item).Elem()

	fields := make([]interface{}, len(fieldIndices))
	for i, idx := range fieldIndices {
		fields[i] = v.Field(idx).Addr().Interface()
	}

	execErr := row.Scan(fields...)

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationGetByID, finalQuery, finalArgs, &item, execErr); err != nil {
		return nil, err
	}

	if execErr != nil {
		return nil, execErr
	}

	if err := c.Hooks.SerializeOne(ctx, hooks.OperationGetByID, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (c *CRUD[T]) Update(ctx context.Context, id any, m T) error {
	if err := c.Hooks.StateProcessor(ctx, hooks.OperationUpdate, id, &m); err != nil {
		return err
	}

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationUpdate, id, &m)

	var query string
	var vals []interface{}

	if skip && customQuery != "" {
		query = customQuery
		vals = customArgs
	} else {
		v := reflect.ValueOf(m)
		t := reflect.TypeOf(m)

		var setClauses []string
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
		query = fmt.Sprintf(
			"UPDATE %s SET %s WHERE id = $%d",
			m.TableName(),
			strings.Join(setClauses, ", "),
			paramIdx,
		)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationUpdate, query, vals)
	if err != nil {
		return err
	}

	_, execErr := c.DB.Exec(ctx, finalQuery, finalArgs...)

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationUpdate, finalQuery, finalArgs, nil, execErr); err != nil {
		return err
	}

	if execErr != nil {
		return execErr
	}

	if err := c.Hooks.SerializeOne(ctx, hooks.OperationUpdate, &m); err != nil {
		return err
	}

	return nil
}

func (c *CRUD[T]) Delete(ctx context.Context, id any) error {
	var zero T

	if err := c.Hooks.StateProcessor(ctx, hooks.OperationDelete, id, nil); err != nil {
		return err
	}

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationDelete, id, nil)

	var query string
	var args []any

	if skip && customQuery != "" {
		query = customQuery
		args = customArgs
	} else {
		query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", zero.TableName())
		args = []any{id}
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationDelete, query, args)
	if err != nil {
		return err
	}

	_, execErr := c.DB.Exec(ctx, finalQuery, finalArgs...)

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationDelete, finalQuery, finalArgs, nil, execErr); err != nil {
		return err
	}

	return execErr
}

type Repository[T any] interface {
	Create(ctx context.Context, m T) error
	GetAll(ctx context.Context) ([]T, error)
	GetByID(ctx context.Context, id any) (*T, error)
	Update(ctx context.Context, id any, m T) error
	Delete(ctx context.Context, id any) error
}
