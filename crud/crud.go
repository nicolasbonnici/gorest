package crud

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/query"
)

type CRUD[T Model] struct {
	DB    database.Database
	Hooks hooks.Hooks[T]
}

// CountMode selects how the total row count backing hydra:totalItems is
// obtained.
type CountMode string

const (
	// CountExact runs a COUNT(*) over the same predicates. Accurate, and the
	// most expensive: on large tables it scans as much as the page query.
	CountExact CountMode = "exact"

	// CountEstimate reads the database's own table statistics instead. Orders of
	// magnitude cheaper, approximate, and only usable for unfiltered listings —
	// any condition or query-scoping hook falls back to CountExact.
	CountEstimate CountMode = "estimate"

	// CountNone omits the total entirely. Callers still get next/previous links.
	CountNone CountMode = "none"
)

type PaginationOptions struct {
	Limit  int
	Offset int
	// IncludeCount requests a total; CountMode decides how it is obtained.
	IncludeCount bool
	// CountMode defaults to CountExact when empty.
	CountMode  CountMode
	Conditions []query.Condition
	OrderBy    []OrderByClause
}

// OrderByClause represents a column ordering for pagination.
type OrderByClause struct {
	Column    string
	Direction query.Order
}

type PaginationResult[T any] struct {
	Items []T
	Total *int
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func IsInvalidIDError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "invalid input syntax for") ||
		strings.Contains(errMsg, "invalid UUID") ||
		strings.Contains(errMsg, "uuid:") ||
		strings.Contains(errMsg, "SQLSTATE 22P02")
}

func New[T Model](db database.Database) *CRUD[T] {
	return &CRUD[T]{
		DB:    db,
		Hooks: hooks.NewNoOpHooks[T](),
	}
}

func NewWithHooks[T Model](db database.Database, h hooks.Hooks[T]) *CRUD[T] {
	return &CRUD[T]{
		DB:    db,
		Hooks: h,
	}
}

func (c *CRUD[T]) Create(ctx context.Context, m T) error {
	// Layer 5: Authorization - Validate field-level write permissions
	if err := c.Hooks.ValidateWrite(ctx, &m); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Layer 5: Authorization - Check resource-level create permission
	if err := c.Hooks.CheckCreate(ctx, &m); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Layer 1: StateProcessor - Validation/enrichment
	if err := c.Hooks.StateProcessor(ctx, hooks.OperationCreate, nil, &m); err != nil {
		return err
	}

	v := reflect.ValueOf(m)
	meta := getFieldMeta(v.Type())

	qb := query.New(c.DB.Dialect()).Insert(m.TableName())

	var cols []string
	var values []any
	idPreSet := false

	for j, idx := range meta.insertIndices {
		tag := meta.insertCols[j]
		if tag == "id" {
			if v.Field(idx).IsZero() {
				continue
			}
			idPreSet = true
		}
		cols = append(cols, tag)
		values = append(values, v.Field(idx).Interface())
	}

	qb = qb.Columns(cols...).Values(values...)

	if c.DB.Dialect().SupportsReturning() && !idPreSet {
		qb = qb.Returning("id")
	}

	queryStr, vals, buildErr := qb.Build()
	if buildErr != nil {
		return fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationCreate, queryStr, vals)
	if err != nil {
		return err
	}

	var execErr error
	var result any
	var createdID any

	if c.DB.Dialect().SupportsReturning() && !idPreSet {
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
		} else if !idPreSet {
			id, err := res.LastInsertId()
			if err != nil {
				execErr = err
			} else {
				createdID = id
				result = id
			}
		}
	}

	if execErr == nil && createdID != nil && !idPreSet && meta.idFieldIndex >= 0 {
		rv := reflect.ValueOf(&m).Elem()
		fieldValue := rv.Field(meta.idFieldIndex)
		if fieldValue.CanSet() {
			switch fieldValue.Kind() {
			case reflect.String:
				if s, ok := createdID.(string); ok {
					fieldValue.SetString(s)
				} else if byteArr, ok := createdID.([16]byte); ok {
					// PostgreSQL UUID returned as [16]byte
					u, err := uuid.FromBytes(byteArr[:])
					if err == nil {
						fieldValue.SetString(u.String())
					} else {
						log.Printf("warning: failed to convert [16]byte to UUID: %v", err)
					}
				} else if byteSlice, ok := createdID.([]byte); ok {
					// PostgreSQL UUID returned as []byte
					if len(byteSlice) == 16 {
						u, err := uuid.FromBytes(byteSlice)
						if err == nil {
							fieldValue.SetString(u.String())
						} else {
							log.Printf("warning: failed to convert []byte to UUID: %v", err)
						}
					} else {
						fieldValue.SetString(string(byteSlice))
					}
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

// Bounds the up-front allocation: limit reaches here straight from the query
// string, so an untrusted value must not reserve unbounded memory.
const maxScanPrealloc = 4096

func scanCapacity(limit int) int {
	if limit <= 0 {
		return 0
	}
	return min(limit, maxScanPrealloc)
}

// capacityHint pre-sizes the result; 0 means unknown. Rows scan directly into
// their slot in the slice because scanning into a local would force each item
// to escape to the heap for the reflect call, then be copied again by append.
//
// scanned counts the rows the database returned, before authorization drops any
// of them, so callers can reason about the underlying result set.
func (c *CRUD[T]) scanRows(ctx context.Context, rows database.Rows, meta *fieldMeta, capacityHint int) ([]T, int, error) {
	results := make([]T, 0, capacityHint)
	scanned := 0

	var zero T
	for rows.Next() {
		scanned++
		// append, not a reslice, so rows beyond capacityHint still grow.
		results = append(results, zero)
		item := &results[len(results)-1]

		sp := meta.acquireScanTargets(reflect.ValueOf(item).Elem())
		err := rows.Scan(*sp...)
		meta.releaseScanTargets(sp)
		if err != nil {
			return nil, scanned, err
		}

		// Layer 5: Authorization - Check resource-level read permission
		if err := c.Hooks.CheckRead(ctx, item); err != nil {
			// Skip unauthorized items (404 behavior - don't disclose existence).
			// Clearing keeps the dropped row out of the slice's spare capacity.
			*item = zero
			results = results[:len(results)-1]
			continue
		}

		// Layer 5: Authorization - Filter forbidden fields
		if err := c.Hooks.FilterRead(ctx, item); err != nil {
			return nil, scanned, fmt.Errorf("authorization failed: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, scanned, err
	}

	return results, scanned, nil
}

func (c *CRUD[T]) GetAll(ctx context.Context) ([]T, error) {
	var zero T
	meta := getFieldMeta(reflect.TypeOf(zero))

	qb := query.New(c.DB.Dialect()).Select(meta.allCols...).From(zero.TableName())
	modifiedBuilder, modified := c.Hooks.ModifySelectQuery(ctx, hooks.OperationGetAll, qb)
	if modified {
		qb = modifiedBuilder
	}

	queryStr, args, buildErr := qb.Build()
	if buildErr != nil {
		return nil, fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetAll, queryStr, args)
	if err != nil {
		return nil, err
	}

	rows, execErr := c.DB.Query(ctx, finalQuery, finalArgs...)
	if execErr != nil {
		_ = c.Hooks.AfterQuery(ctx, hooks.OperationGetAll, finalQuery, finalArgs, nil, execErr)
		return nil, execErr
	}
	defer rows.Close()

	results, _, err := c.scanRows(ctx, rows, meta, 0)
	if err != nil {
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
	meta := getFieldMeta(reflect.TypeOf(zero))

	qb := query.New(c.DB.Dialect()).Select(meta.allCols...).From(zero.TableName())

	modifiedBuilder, modified := c.Hooks.ModifySelectQuery(ctx, hooks.OperationGetAll, qb)
	if modified {
		qb = modifiedBuilder
	}

	for _, cond := range opts.Conditions {
		qb = qb.Where(cond)
	}

	for _, order := range opts.OrderBy {
		qb = qb.OrderBy(order.Column, order.Direction)
	}

	qb = qb.Limit(opts.Limit).Offset(opts.Offset)

	queryStr, args, buildErr := qb.Build()
	if buildErr != nil {
		return nil, fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetAll, queryStr, args)
	if err != nil {
		return nil, err
	}

	rows, execErr := c.DB.Query(ctx, finalQuery, finalArgs...)
	if execErr != nil {
		_ = c.Hooks.AfterQuery(ctx, hooks.OperationGetAll, finalQuery, finalArgs, nil, execErr)
		return nil, execErr
	}
	defer rows.Close()

	results, scanned, scanErr := c.scanRows(ctx, rows, meta, scanCapacity(opts.Limit))
	if scanErr != nil {
		return nil, scanErr
	}

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationGetAll, finalQuery, finalArgs, results, nil); err != nil {
		return nil, err
	}

	if err := c.Hooks.SerializeMany(ctx, hooks.OperationGetAll, &results); err != nil {
		return nil, err
	}

	// Counted after the page is fetched: a short page already reveals the total,
	// which saves the count query entirely.
	total, err := c.countTotal(ctx, zero.TableName(), opts, modified, scanned)
	if err != nil {
		return nil, err
	}

	return &PaginationResult[T]{
		Items: results,
		Total: total,
	}, nil
}

// countTotal resolves hydra:totalItems for a page. scoped reports whether a hook
// narrowed the query, which rules out the table-wide estimate.
func (c *CRUD[T]) countTotal(ctx context.Context, table string, opts PaginationOptions, scoped bool, scanned int) (*int, error) {
	if !opts.IncludeCount || opts.CountMode == CountNone {
		return nil, nil
	}

	// The database returned fewer rows than asked for, so this page is the last
	// one and the total follows from the offset. Exact, and free.
	if opts.Limit <= 0 || scanned < opts.Limit {
		total := opts.Offset + scanned
		return &total, nil
	}

	if opts.CountMode == CountEstimate && len(opts.Conditions) == 0 && !scoped {
		if total, ok := c.estimateTotal(ctx, table); ok {
			return total, nil
		}
	}

	return c.exactTotal(ctx, table, opts)
}

func (c *CRUD[T]) exactTotal(ctx context.Context, table string, opts PaginationOptions) (*int, error) {
	countBuilder := query.New(c.DB.Dialect()).Select("COUNT(*)").From(table)
	countBuilder, _ = c.Hooks.ModifySelectQuery(ctx, hooks.OperationGetAll, countBuilder)
	for _, cond := range opts.Conditions {
		countBuilder = countBuilder.Where(cond)
	}

	countQuery, countArgs, countErr := countBuilder.Build()
	if countErr != nil {
		return nil, fmt.Errorf("count query build failed: %w", countErr)
	}

	var count int
	if err := c.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&count); err != nil {
		return nil, err
	}
	return &count, nil
}

// estimateTotal reads the database's table statistics. ok is false whenever the
// figure cannot be trusted — dialect without support, query failure, or a table
// the database has no statistics for — leaving the caller to count exactly.
func (c *CRUD[T]) estimateTotal(ctx context.Context, table string) (*int, bool) {
	estimator, ok := c.DB.Dialect().(database.RowEstimator)
	if !ok {
		return nil, false
	}

	estimateQuery, estimateArgs, ok := estimator.EstimateRowsQuery(table)
	if !ok {
		return nil, false
	}

	var estimate int64
	if err := c.DB.QueryRow(ctx, estimateQuery, estimateArgs...).Scan(&estimate); err != nil {
		return nil, false
	}

	if estimate < 0 {
		return nil, false
	}

	total := int(estimate)
	return &total, true
}

func (c *CRUD[T]) GetByID(ctx context.Context, id any) (*T, error) {
	var item T
	meta := getFieldMeta(reflect.TypeOf(item))

	qb := query.New(c.DB.Dialect()).Select(meta.allCols...).From(item.TableName()).Where(query.Eq("id", id))
	modifiedBuilder, modified := c.Hooks.ModifySelectQuery(ctx, hooks.OperationGetByID, qb)
	if modified {
		qb = modifiedBuilder
	}

	queryStr, args, buildErr := qb.Build()
	if buildErr != nil {
		return nil, fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetByID, queryStr, args)
	if err != nil {
		return nil, err
	}

	row := c.DB.QueryRow(ctx, finalQuery, finalArgs...)

	sp := meta.acquireScanTargets(reflect.ValueOf(&item).Elem())
	execErr := row.Scan(*sp...)
	meta.releaseScanTargets(sp)

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationGetByID, finalQuery, finalArgs, &item, execErr); err != nil {
		return nil, err
	}

	if execErr != nil {
		return nil, execErr
	}

	// Layer 5: Authorization - Check resource-level read permission
	if err := c.Hooks.CheckRead(ctx, &item); err != nil {
		// Return 404 for unauthorized reads (don't disclose existence)
		return nil, sql.ErrNoRows
	}

	// Layer 5: Authorization - Filter forbidden fields
	if err := c.Hooks.FilterRead(ctx, &item); err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if err := c.Hooks.SerializeOne(ctx, hooks.OperationGetByID, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// GetByIDs resolves many rows in a single round-trip. It runs the GetByID hooks
// so query scoping (multi-tenancy, soft deletes) stays identical to fetching the
// same rows one at a time.
func (c *CRUD[T]) GetByIDs(ctx context.Context, ids []any) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}

	var zero T
	meta := getFieldMeta(reflect.TypeOf(zero))

	qb := query.New(c.DB.Dialect()).
		Select(meta.allCols...).
		From(zero.TableName()).
		Where(query.In("id", ids...))

	modifiedBuilder, modified := c.Hooks.ModifySelectQuery(ctx, hooks.OperationGetByID, qb)
	if modified {
		qb = modifiedBuilder
	}

	queryStr, args, buildErr := qb.Build()
	if buildErr != nil {
		return nil, fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationGetByID, queryStr, args)
	if err != nil {
		return nil, err
	}

	rows, execErr := c.DB.Query(ctx, finalQuery, finalArgs...)
	if execErr != nil {
		_ = c.Hooks.AfterQuery(ctx, hooks.OperationGetByID, finalQuery, finalArgs, nil, execErr)
		return nil, execErr
	}
	defer rows.Close()

	results, _, err := c.scanRows(ctx, rows, meta, scanCapacity(len(ids)))
	if err != nil {
		return nil, err
	}

	if err := c.Hooks.AfterQuery(ctx, hooks.OperationGetByID, finalQuery, finalArgs, results, nil); err != nil {
		return nil, err
	}

	if err := c.Hooks.SerializeMany(ctx, hooks.OperationGetByID, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (c *CRUD[T]) Update(ctx context.Context, id any, m T) error {
	// Layer 5: Authorization - Validate field-level write permissions
	if err := c.Hooks.ValidateWrite(ctx, &m); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Layer 5: Authorization - Check resource-level update permission
	if err := c.Hooks.CheckUpdate(ctx, id, &m); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Layer 1: StateProcessor - Validation/enrichment
	if err := c.Hooks.StateProcessor(ctx, hooks.OperationUpdate, id, &m); err != nil {
		return err
	}

	v := reflect.ValueOf(m)
	meta := getFieldMeta(v.Type())

	qb := query.New(c.DB.Dialect()).Update(m.TableName())
	for j, idx := range meta.updateIndices {
		qb = qb.Set(meta.updateCols[j], v.Field(idx).Interface())
	}
	qb = qb.Where(query.Eq("id", id))

	modifiedBuilder, modified := c.Hooks.ModifyUpdateQuery(ctx, hooks.OperationUpdate, id, &m, qb)
	if modified {
		qb = modifiedBuilder
	}

	queryStr, vals, buildErr := qb.Build()
	if buildErr != nil {
		return fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationUpdate, queryStr, vals)
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

	// Layer 5: Authorization - Check resource-level delete permission
	if err := c.Hooks.CheckDelete(ctx, id); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Layer 1: StateProcessor - Validation/enrichment
	if err := c.Hooks.StateProcessor(ctx, hooks.OperationDelete, id, nil); err != nil {
		return err
	}

	qb := query.New(c.DB.Dialect()).Delete(zero.TableName()).Where(query.Eq("id", id))

	modifiedBuilder, modified := c.Hooks.ModifyDeleteQuery(ctx, hooks.OperationDelete, id, qb)
	if modified {
		qb = modifiedBuilder
	}

	queryStr, args, buildErr := qb.Build()
	if buildErr != nil {
		return fmt.Errorf("query build failed: %w", buildErr)
	}

	finalQuery, finalArgs, err := c.Hooks.BeforeQuery(ctx, hooks.OperationDelete, queryStr, args)
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
