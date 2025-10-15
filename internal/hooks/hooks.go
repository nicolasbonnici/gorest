package hooks

import "context"

type Operation string

const (
	OperationCreate  Operation = "CREATE"
	OperationGetAll  Operation = "GET_ALL"
	OperationGetByID Operation = "GET_BY_ID"
	OperationUpdate  Operation = "UPDATE"
	OperationDelete  Operation = "DELETE"
)

type StateProcessor[T any] interface {
	StateProcessor(ctx context.Context, operation Operation, id any, model *T) error
}

type SQLQueryListener[T any] interface {
	BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error)
	AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error
}

type SQLQueryOverride[T any] interface {
	OverrideQuery(ctx context.Context, operation Operation, id any, model *T) (query string, args []any, skip bool)
}

type Serializer[T any] interface {
	SerializeOne(ctx context.Context, operation Operation, model *T) error
	SerializeMany(ctx context.Context, operation Operation, models *[]T) error
}

type Hooks[T any] interface {
	StateProcessor[T]
	SQLQueryListener[T]
	SQLQueryOverride[T]
	Serializer[T]
}

type NoOpHooks[T any] struct{}

func (h NoOpHooks[T]) StateProcessor(ctx context.Context, operation Operation, id any, model *T) error {
	return nil
}

func (h NoOpHooks[T]) BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error) {
	return query, args, nil
}

func (h NoOpHooks[T]) AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error {
	return nil
}

func (h NoOpHooks[T]) OverrideQuery(ctx context.Context, operation Operation, id any, model *T) (query string, args []any, skip bool) {
	return "", nil, false
}

func (h NoOpHooks[T]) SerializeOne(ctx context.Context, operation Operation, model *T) error {
	return nil
}

func (h NoOpHooks[T]) SerializeMany(ctx context.Context, operation Operation, models *[]T) error {
	return nil
}
