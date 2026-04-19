package hooks

import (
	"context"

	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/rbac"
)

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

type SQLQueryBuilderModifier[T any] interface {
	ModifySelectQuery(ctx context.Context, operation Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool)
	ModifyUpdateQuery(ctx context.Context, operation Operation, id any, model *T, builder *query.UpdateBuilder) (*query.UpdateBuilder, bool)
	ModifyDeleteQuery(ctx context.Context, operation Operation, id any, builder *query.DeleteBuilder) (*query.DeleteBuilder, bool)
}

type Serializer[T any] interface {
	SerializeOne(ctx context.Context, operation Operation, model *T) error
	SerializeMany(ctx context.Context, operation Operation, models *[]T) error
}

// Authorization is the 5th hook layer for RBAC (mandatory)
type Authorization[T any] interface {
	CheckCreate(ctx context.Context, model *T) error
	CheckRead(ctx context.Context, model *T) error
	CheckUpdate(ctx context.Context, id any, model *T) error
	CheckDelete(ctx context.Context, id any) error
	FilterRead(ctx context.Context, model *T) error
	ValidateWrite(ctx context.Context, model *T) error
	GetVoter() rbac.Voter
}

type Hooks[T any] interface {
	Authorization[T]           // Layer 5 - RBAC (NEW, MANDATORY)
	StateProcessor[T]          // Layer 1
	SQLQueryListener[T]        // Layer 2
	SQLQueryBuilderModifier[T] // Layer 3
	Serializer[T]              // Layer 4
}

// DefaultAuthorization provides tag-based RBAC authorization
type DefaultAuthorization[T any] struct {
	voter rbac.Voter
}

// NewDefaultAuthorization creates a new DefaultAuthorization with the given config
func NewDefaultAuthorization[T any](config rbac.Config) *DefaultAuthorization[T] {
	voter, err := rbac.NewVoter(config)
	if err != nil {
		// Fall back to default config if validation fails
		voter, _ = rbac.NewVoter(rbac.DefaultConfig())
	}
	return &DefaultAuthorization[T]{
		voter: voter,
	}
}

// CheckCreate verifies if the user can create this resource
func (a *DefaultAuthorization[T]) CheckCreate(ctx context.Context, model *T) error {
	// By default, allow create if user has any role (can be overridden)
	roles, ok := rbac.GetRoles(ctx)
	if !ok || len(roles) == 0 {
		if a.voter.GetConfig().DefaultPolicy == rbac.DenyAll {
			return rbac.ErrPermissionDenied
		}
	}
	return nil
}

// CheckRead verifies if the user can read this resource
func (a *DefaultAuthorization[T]) CheckRead(ctx context.Context, model *T) error {
	// By default, allow read if user has any role (can be overridden)
	roles, ok := rbac.GetRoles(ctx)
	if !ok || len(roles) == 0 {
		if a.voter.GetConfig().DefaultPolicy == rbac.DenyAll {
			return rbac.ErrPermissionDenied
		}
	}
	return nil
}

// CheckUpdate verifies if the user can update this resource
func (a *DefaultAuthorization[T]) CheckUpdate(ctx context.Context, id any, model *T) error {
	// By default, allow update if user has any role (can be overridden)
	roles, ok := rbac.GetRoles(ctx)
	if !ok || len(roles) == 0 {
		if a.voter.GetConfig().DefaultPolicy == rbac.DenyAll {
			return rbac.ErrPermissionDenied
		}
	}
	return nil
}

// CheckDelete verifies if the user can delete this resource
func (a *DefaultAuthorization[T]) CheckDelete(ctx context.Context, id any) error {
	// By default, allow delete if user has any role (can be overridden)
	roles, ok := rbac.GetRoles(ctx)
	if !ok || len(roles) == 0 {
		if a.voter.GetConfig().DefaultPolicy == rbac.DenyAll {
			return rbac.ErrPermissionDenied
		}
	}
	return nil
}

// FilterRead filters the model to remove fields the user cannot read
func (a *DefaultAuthorization[T]) FilterRead(ctx context.Context, model *T) error {
	filtered, err := a.voter.FilterRead(ctx, model)
	if err != nil {
		return err
	}
	// Copy filtered fields back to model
	*model = *(filtered.(*T))
	return nil
}

// ValidateWrite validates that all non-zero fields can be written by the user
func (a *DefaultAuthorization[T]) ValidateWrite(ctx context.Context, model *T) error {
	return a.voter.ValidateWrite(ctx, model)
}

// GetVoter returns the underlying voter
func (a *DefaultAuthorization[T]) GetVoter() rbac.Voter {
	return a.voter
}

type NoOpHooks[T any] struct {
	*DefaultAuthorization[T]
}

// NewNoOpHooks creates a NoOpHooks with permissive RBAC configuration
// This is suitable for testing and scenarios where authorization is not needed
func NewNoOpHooks[T any]() *NoOpHooks[T] {
	config := rbac.DefaultConfig()
	config.DefaultPolicy = rbac.AllowAll // Allow operations without roles
	config.DefaultFieldPolicy = "allow"  // Allow fields without rbac tags
	config.StrictValidation = false      // Don't validate zero values
	return &NoOpHooks[T]{
		DefaultAuthorization: NewDefaultAuthorization[T](config),
	}
}

// NewNoOpHooksWithConfig creates a NoOpHooks with custom RBAC configuration
func NewNoOpHooksWithConfig[T any](config rbac.Config) *NoOpHooks[T] {
	return &NoOpHooks[T]{
		DefaultAuthorization: NewDefaultAuthorization[T](config),
	}
}

func (h *NoOpHooks[T]) StateProcessor(ctx context.Context, operation Operation, id any, model *T) error {
	return nil
}

func (h *NoOpHooks[T]) BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error) {
	return query, args, nil
}

func (h *NoOpHooks[T]) AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error {
	return nil
}

func (h *NoOpHooks[T]) ModifySelectQuery(ctx context.Context, operation Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
	return builder, false
}

func (h *NoOpHooks[T]) ModifyUpdateQuery(ctx context.Context, operation Operation, id any, model *T, builder *query.UpdateBuilder) (*query.UpdateBuilder, bool) {
	return builder, false
}

func (h *NoOpHooks[T]) ModifyDeleteQuery(ctx context.Context, operation Operation, id any, builder *query.DeleteBuilder) (*query.DeleteBuilder, bool) {
	return builder, false
}

func (h *NoOpHooks[T]) SerializeOne(ctx context.Context, operation Operation, model *T) error {
	return nil
}

func (h *NoOpHooks[T]) SerializeMany(ctx context.Context, operation Operation, models *[]T) error {
	return nil
}
