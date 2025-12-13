package crud

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
)

type CRUD[T Model] struct {
	DB    database.Database
	Hooks hooks.Hooks[T]
}

type PaginationOptions struct {
	Limit         int
	Offset        int
	IncludeCount  bool
	WhereClause   string
	WhereArgs     []interface{}
	OrderByClause string
}

type PaginationResult[T any] struct {
	Items []T
	Total *int
}

func New[T Model](db database.Database) *CRUD[T] {
	return &CRUD[T]{
		DB:    db,
		Hooks: hooks.NoOpHooks[T]{},
	}
}

func NewWithHooks[T Model](db database.Database, h hooks.Hooks[T]) *CRUD[T] {
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
			placeholders = append(placeholders, c.DB.Dialect().Placeholder(len(vals)))
		}

		returning := ""
		if c.DB.Dialect().SupportsReturning() {
			returning = " " + c.DB.Dialect().ReturningClause()
		}

		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)%s",
			m.TableName(),
			strings.Join(cols, ", "),
			strings.Join(placeholders, ", "),
			returning,
		)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationCreate, query, vals)
	if err != nil {
		return err
	}

	var execErr error
	var result any
	var createdID any

	if c.DB.Dialect().SupportsReturning() {
		row := c.DB.QueryRow(ctx, finalQuery, finalArgs...)
		scanErr := row.Scan(&createdID)
		if scanErr != nil {
			execErr = scanErr
		} else {
			result = createdID
		}
	} else {
		res, err := c.DB.Exec(ctx, finalQuery, finalArgs...)
		if err != nil {
			execErr = err
		} else {
			id, err := res.LastInsertId()
			if err != nil {
				execErr = err
			} else {
				createdID = id
				result = id
			}
		}
	}

	if execErr == nil && createdID != nil {
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

func (c *CRUD[T]) GetAllPaginated(ctx context.Context, opts PaginationOptions) (*PaginationResult[T], error) {
	var zero T

	customQuery, customArgs, skip := c.Hooks.OverrideQuery(ctx, hooks.OperationGetAll, nil, nil)

	var baseQuery string
	var args []any
	var fieldIndices []int

	if skip && customQuery != "" {
		baseQuery = customQuery
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

		baseQuery = fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), zero.TableName())
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

	var total *int
	if opts.IncludeCount {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", zero.TableName())
		if opts.WhereClause != "" {
			countQuery += " " + opts.WhereClause
		}
		var count int
		if err := c.DB.QueryRow(ctx, countQuery, opts.WhereArgs...).Scan(&count); err != nil {
			return nil, err
		}
		total = &count
	}

	query := baseQuery
	if opts.WhereClause != "" {
		query += " " + opts.WhereClause
	}
	if opts.OrderByClause != "" {
		query += " " + opts.OrderByClause
	}
	limitOffsetClause := c.DB.Dialect().LimitOffset(opts.Limit, opts.Offset)
	query += " " + limitOffsetClause

	args = append(args, opts.WhereArgs...)

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

	return &PaginationResult[T]{
		Items: results,
		Total: total,
	}, nil
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

		query = fmt.Sprintf("SELECT %s FROM %s WHERE id = %s", strings.Join(cols, ", "), item.TableName(), c.DB.Dialect().Placeholder(1))
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

func (c *CRUD[T]) GetByIDs(ctx context.Context, ids []any) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}

	var zero T
	t := reflect.TypeOf(zero)

	var cols []string
	var fieldIndices []int
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			cols = append(cols, tag)
			fieldIndices = append(fieldIndices, i)
		}
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = c.DB.Dialect().Placeholder(i + 1)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id IN (%s)",
		strings.Join(cols, ", "),
		zero.TableName(),
		strings.Join(placeholders, ", "),
	)

	rows, err := c.DB.Query(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []T
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

		items = append(items, item)
	}

	return items, nil
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
			setClauses = append(setClauses, fmt.Sprintf("%s = %s", tag, c.DB.Dialect().Placeholder(paramIdx)))
			vals = append(vals, v.Field(i).Interface())
			paramIdx++
		}

		vals = append(vals, id)
		query = fmt.Sprintf(
			"UPDATE %s SET %s WHERE id = %s",
			m.TableName(),
			strings.Join(setClauses, ", "),
			c.DB.Dialect().Placeholder(paramIdx),
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
		query = fmt.Sprintf("DELETE FROM %s WHERE id = %s", zero.TableName(), c.DB.Dialect().Placeholder(1))
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
