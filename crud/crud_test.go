package crud

import (
	"context"
	"errors"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
)

type testModel struct {
	ID        int64  `db:"id"`
	Name      string `db:"name"`
	Email     string `db:"email"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

func (m testModel) TableName() string {
	return "test_table"
}

type mockDialect struct {
	name              string
	supportsReturning bool
}

func (m *mockDialect) Placeholder(n int) string {
	switch m.name {
	case "postgres":
		return "$" + string(rune('0'+n))
	case "mysql", "sqlite":
		return "?"
	default:
		return "?"
	}
}

func (m *mockDialect) SupportsReturning() bool {
	return m.supportsReturning
}

func (m *mockDialect) ReturningClause(cols ...string) string {
	if m.supportsReturning {
		return "RETURNING id"
	}
	return ""
}

func (m *mockDialect) LimitOffset(limit, offset int) string {
	return "LIMIT ? OFFSET ?"
}

func (m *mockDialect) QuoteIdentifier(name string) string {
	return name
}

func (m *mockDialect) MapType(dbType string) string {
	return dbType
}

func (m *mockDialect) CaseInsensitiveLike() string {
	return "LOWER"
}

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

type mockRows struct {
	scanFunc  func(dest ...interface{}) error
	nextFunc  func() bool
	closeErr  error
	rowErr    error
	callCount int
}

func (m *mockRows) Next() bool {
	return m.nextFunc()
}

func (m *mockRows) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

func (m *mockRows) Close() error {
	return m.closeErr
}

func (m *mockRows) Err() error {
	return m.rowErr
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
	insertErr    error
	affectedErr  error
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, m.insertErr
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.affectedErr
}

type mockDatabase struct {
	dialect      database.Dialect
	queryRowFunc func(ctx context.Context, query string, args ...interface{}) database.Row
	queryFunc    func(ctx context.Context, query string, args ...interface{}) (database.Rows, error)
	execFunc     func(ctx context.Context, query string, args ...interface{}) (database.Result, error)
}

func (m *mockDatabase) Connect(ctx context.Context, dsn string) error {
	return nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return nil
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return &mockResult{}, nil
}

func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) {
	return nil, nil
}

func (m *mockDatabase) Dialect() database.Dialect {
	return m.dialect
}

func (m *mockDatabase) DriverName() string {
	return ""
}

func (m *mockDatabase) Introspector() database.SchemaIntrospector {
	return nil
}

type mockHooks struct {
	hooks.NoOpHooks[testModel]
	stateProcessorFunc func(ctx context.Context, operation hooks.Operation, id any, model *testModel) error
	beforeQueryFunc    func(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error)
	afterQueryFunc     func(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error
	overrideQueryFunc  func(ctx context.Context, operation hooks.Operation, id any, model *testModel) (query string, args []any, skip bool)
	serializeOneFunc   func(ctx context.Context, operation hooks.Operation, model *testModel) error
	serializeManyFunc  func(ctx context.Context, operation hooks.Operation, models *[]testModel) error
}

func (m *mockHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, model *testModel) error {
	if m.stateProcessorFunc != nil {
		return m.stateProcessorFunc(ctx, operation, id, model)
	}
	return nil
}

func (m *mockHooks) BeforeQuery(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
	if m.beforeQueryFunc != nil {
		return m.beforeQueryFunc(ctx, operation, query, args)
	}
	return query, args, nil
}

func (m *mockHooks) AfterQuery(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
	if m.afterQueryFunc != nil {
		return m.afterQueryFunc(ctx, operation, query, args, result, err)
	}
	return nil
}

func (m *mockHooks) OverrideQuery(ctx context.Context, operation hooks.Operation, id any, model *testModel) (query string, args []any, skip bool) {
	if m.overrideQueryFunc != nil {
		return m.overrideQueryFunc(ctx, operation, id, model)
	}
	return "", nil, false
}

func (m *mockHooks) SerializeOne(ctx context.Context, operation hooks.Operation, model *testModel) error {
	if m.serializeOneFunc != nil {
		return m.serializeOneFunc(ctx, operation, model)
	}
	return nil
}

func (m *mockHooks) SerializeMany(ctx context.Context, operation hooks.Operation, models *[]testModel) error {
	if m.serializeManyFunc != nil {
		return m.serializeManyFunc(ctx, operation, models)
	}
	return nil
}

func TestNew(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
	}

	crud := New[testModel](db)

	if crud == nil {
		t.Fatal("expected non-nil CRUD instance")
	}

	if crud.DB != db {
		t.Error("expected DB to be set")
	}

	if crud.Hooks == nil {
		t.Error("expected Hooks to be initialized")
	}
}

func TestNewWithHooks(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
	}
	customHooks := &mockHooks{}

	crud := NewWithHooks[testModel](db, customHooks)

	if crud == nil {
		t.Fatal("expected non-nil CRUD instance")
	}

	if crud.DB != db {
		t.Error("expected DB to be set")
	}

	if crud.Hooks != customHooks {
		t.Error("expected custom hooks to be set")
	}
}

func TestCreate_Postgres(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if ptr, ok := dest[0].(*int64); ok {
						*ptr = 123
					} else if ptr, ok := dest[0].(*interface{}); ok {
						*ptr = int64(123)
					}
					return nil
				},
			}
		},
	}

	crud := New[testModel](db)
	model := testModel{
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := crud.Create(context.Background(), model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if model.ID != 0 {
		t.Errorf("expected ID 0 (model passed by value), got %d", model.ID)
	}
}

func TestCreate_MySQL(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "mysql", supportsReturning: false},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{lastInsertID: 456}, nil
		},
	}

	crud := New[testModel](db)
	model := testModel{
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := crud.Create(context.Background(), model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreate_SQLite(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "sqlite", supportsReturning: false},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{lastInsertID: 789}, nil
		},
	}

	crud := New[testModel](db)
	model := testModel{
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := crud.Create(context.Background(), model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreate_PostgresError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					return errors.New("database error")
				},
			}
		},
	}

	crud := New[testModel](db)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_MySQLError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "mysql", supportsReturning: false},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return nil, errors.New("insert failed")
		},
	}

	crud := New[testModel](db)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_WithHooks(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if ptr, ok := dest[0].(*int64); ok {
						*ptr = 100
					} else if ptr, ok := dest[0].(*interface{}); ok {
						*ptr = int64(100)
					}
					return nil
				},
			}
		},
	}

	stateProcessorCalled := false
	beforeQueryCalled := false
	afterQueryCalled := false
	serializeOneCalled := false

	customHooks := &mockHooks{
		stateProcessorFunc: func(ctx context.Context, operation hooks.Operation, id any, model *testModel) error {
			stateProcessorCalled = true
			return nil
		},
		beforeQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
			beforeQueryCalled = true
			return query, args, nil
		},
		afterQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
			afterQueryCalled = true
			return nil
		},
		serializeOneFunc: func(ctx context.Context, operation hooks.Operation, model *testModel) error {
			serializeOneCalled = true
			return nil
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !stateProcessorCalled {
		t.Error("expected StateProcessor to be called")
	}
	if !beforeQueryCalled {
		t.Error("expected BeforeQuery to be called")
	}
	if !afterQueryCalled {
		t.Error("expected AfterQuery to be called")
	}
	if !serializeOneCalled {
		t.Error("expected SerializeOne to be called")
	}
}

func TestGetByID_Success(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = 123
					*dest[1].(*string) = "John Doe"
					*dest[2].(*string) = "john@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetByID(context.Background(), 123)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
	if result.Name != "John Doe" {
		t.Errorf("expected Name 'John Doe', got '%s'", result.Name)
	}
	if result.Email != "john@example.com" {
		t.Errorf("expected Email 'john@example.com', got '%s'", result.Email)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					return errors.New("sql: no rows in result set")
				},
			}
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetByID(context.Background(), 999)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != nil {
		t.Error("expected nil result")
	}
}

func TestGetAll_Success(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 3
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = int64(callCount)
					*dest[1].(*string) = "User" + string(rune('0'+callCount))
					*dest[2].(*string) = "user" + string(rune('0'+callCount)) + "@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	results, err := crud.GetAll(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestGetAll_Empty(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					return false
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	results, err := crud.GetAll(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetAll_QueryError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return nil, errors.New("query failed")
		},
	}

	crud := New[testModel](db)
	results, err := crud.GetAll(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if results != nil {
		t.Error("expected nil results")
	}
}

func TestGetAllPaginated_WithCount(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int) = 100
					return nil
				},
			}
		},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 10
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = int64(callCount)
					*dest[1].(*string) = "User"
					*dest[2].(*string) = "user@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		Offset:       0,
		IncludeCount: true,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Total == nil {
		t.Fatal("expected non-nil total count")
	}

	if *result.Total != 100 {
		t.Errorf("expected total 100, got %d", *result.Total)
	}

	if len(result.Items) != 10 {
		t.Errorf("expected 10 items, got %d", len(result.Items))
	}
}

func TestGetAllPaginated_WithoutCount(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 5
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = int64(callCount)
					*dest[1].(*string) = "User"
					*dest[2].(*string) = "user@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        5,
		Offset:       0,
		IncludeCount: false,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != nil {
		t.Error("expected nil total count")
	}

	if len(result.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(result.Items))
	}
}

func TestGetAllPaginated_WithWhereClause(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 2
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = int64(callCount)
					*dest[1].(*string) = "Active User"
					*dest[2].(*string) = "active@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		Offset:       0,
		WhereClause:  "WHERE status = ?",
		WhereArgs:    []interface{}{"active"},
		IncludeCount: false,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetAllPaginated_WithOrderBy(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 3
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = int64(callCount)
					*dest[1].(*string) = "User"
					*dest[2].(*string) = "user@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:         10,
		Offset:        0,
		OrderByClause: "ORDER BY name DESC",
		IncludeCount:  false,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
}

func TestUpdate_Success(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{rowsAffected: 1}, nil
		},
	}

	crud := New[testModel](db)
	model := testModel{
		Name:  "Updated Name",
		Email: "updated@example.com",
	}

	err := crud.Update(context.Background(), 123, model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdate_Error(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return nil, errors.New("update failed")
		},
	}

	crud := New[testModel](db)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Update(context.Background(), 123, model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdate_WithHooks(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{rowsAffected: 1}, nil
		},
	}

	stateProcessorCalled := false
	beforeQueryCalled := false
	afterQueryCalled := false
	serializeOneCalled := false

	customHooks := &mockHooks{
		stateProcessorFunc: func(ctx context.Context, operation hooks.Operation, id any, model *testModel) error {
			stateProcessorCalled = true
			return nil
		},
		beforeQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
			beforeQueryCalled = true
			return query, args, nil
		},
		afterQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
			afterQueryCalled = true
			return nil
		},
		serializeOneFunc: func(ctx context.Context, operation hooks.Operation, model *testModel) error {
			serializeOneCalled = true
			return nil
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Update(context.Background(), 123, model)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !stateProcessorCalled {
		t.Error("expected StateProcessor to be called")
	}
	if !beforeQueryCalled {
		t.Error("expected BeforeQuery to be called")
	}
	if !afterQueryCalled {
		t.Error("expected AfterQuery to be called")
	}
	if !serializeOneCalled {
		t.Error("expected SerializeOne to be called")
	}
}

func TestDelete_Success(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{rowsAffected: 1}, nil
		},
	}

	crud := New[testModel](db)
	err := crud.Delete(context.Background(), 123)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return nil, errors.New("delete failed")
		},
	}

	crud := New[testModel](db)
	err := crud.Delete(context.Background(), 123)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDelete_WithHooks(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
			return &mockResult{rowsAffected: 1}, nil
		},
	}

	stateProcessorCalled := false
	beforeQueryCalled := false
	afterQueryCalled := false

	customHooks := &mockHooks{
		stateProcessorFunc: func(ctx context.Context, operation hooks.Operation, id any, model *testModel) error {
			stateProcessorCalled = true
			return nil
		},
		beforeQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
			beforeQueryCalled = true
			return query, args, nil
		},
		afterQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
			afterQueryCalled = true
			return nil
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	err := crud.Delete(context.Background(), 123)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !stateProcessorCalled {
		t.Error("expected StateProcessor to be called")
	}
	if !beforeQueryCalled {
		t.Error("expected BeforeQuery to be called")
	}
	if !afterQueryCalled {
		t.Error("expected AfterQuery to be called")
	}
}

func TestCreate_HooksStateProcessorError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
	}

	customHooks := &mockHooks{
		stateProcessorFunc: func(ctx context.Context, operation hooks.Operation, id any, model *testModel) error {
			return errors.New("state processor failed")
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "state processor failed" {
		t.Errorf("expected 'state processor failed', got '%s'", err.Error())
	}
}

func TestCreate_HooksBeforeQueryError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
	}

	customHooks := &mockHooks{
		beforeQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
			return "", nil, errors.New("before query failed")
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_HooksAfterQueryError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if ptr, ok := dest[0].(*int64); ok {
						*ptr = 123
					} else if ptr, ok := dest[0].(*interface{}); ok {
						*ptr = int64(123)
					}
					return nil
				},
			}
		},
	}

	customHooks := &mockHooks{
		afterQueryFunc: func(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
			return errors.New("after query failed")
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_HooksSerializeOneError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if ptr, ok := dest[0].(*int64); ok {
						*ptr = 123
					} else if ptr, ok := dest[0].(*interface{}); ok {
						*ptr = int64(123)
					}
					return nil
				},
			}
		},
	}

	customHooks := &mockHooks{
		serializeOneFunc: func(ctx context.Context, operation hooks.Operation, model *testModel) error {
			return errors.New("serialize failed")
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	model := testModel{Name: "Test", Email: "test@example.com"}

	err := crud.Create(context.Background(), model)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetAll_HooksOverrideQuery(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 1
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = 999
					*dest[1].(*string) = "Override"
					*dest[2].(*string) = "override@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
			}, nil
		},
	}

	customHooks := &mockHooks{
		overrideQueryFunc: func(ctx context.Context, operation hooks.Operation, id any, model *testModel) (query string, args []any, skip bool) {
			return "SELECT * FROM custom_table", []any{}, true
		},
	}

	crud := NewWithHooks[testModel](db, customHooks)
	results, err := crud.GetAll(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestGetAll_RowsError(t *testing.T) {
	callCount := 0
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
			return &mockRows{
				nextFunc: func() bool {
					callCount++
					return callCount <= 1
				},
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*int64) = 1
					*dest[1].(*string) = "User"
					*dest[2].(*string) = "user@example.com"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*string) = "2024-01-01"
					return nil
				},
				rowErr: errors.New("rows error"),
			}, nil
		},
	}

	crud := New[testModel](db)
	results, err := crud.GetAll(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if results != nil {
		t.Error("expected nil results")
	}
}

func TestGetAllPaginated_CountError(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres", supportsReturning: true},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					return errors.New("count query failed")
				},
			}
		},
	}

	crud := New[testModel](db)
	result, err := crud.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		Offset:       0,
		IncludeCount: true,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != nil {
		t.Error("expected nil result")
	}
}

func TestMultipleDialects_PlaceholderDifferences(t *testing.T) {
	tests := []struct {
		name              string
		dialectName       string
		supportsReturning bool
		expectedPattern   string
	}{
		{"Postgres", "postgres", true, "RETURNING"},
		{"MySQL", "mysql", false, "VALUES"},
		{"SQLite", "sqlite", false, "VALUES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedQuery string
			db := &mockDatabase{
				dialect: &mockDialect{name: tt.dialectName, supportsReturning: tt.supportsReturning},
			}

			if tt.supportsReturning {
				db.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) database.Row {
					capturedQuery = query
					return &mockRow{
						scanFunc: func(dest ...interface{}) error {
							if ptr, ok := dest[0].(*int64); ok {
								*ptr = 1
							} else if ptr, ok := dest[0].(*interface{}); ok {
								*ptr = int64(1)
							}
							return nil
						},
					}
				}
			} else {
				db.execFunc = func(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
					capturedQuery = query
					return &mockResult{lastInsertID: 1}, nil
				}
			}

			crud := New[testModel](db)
			model := testModel{Name: "Test", Email: "test@example.com"}
			crud.Create(context.Background(), model)

			if capturedQuery == "" {
				t.Error("expected query to be captured")
			}
		})
	}
}
