package hooks

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/query"
)

func TestOperationConstants(t *testing.T) {
	tests := []struct {
		name     string
		op       Operation
		expected string
	}{
		{"Create operation", OperationCreate, "CREATE"},
		{"GetAll operation", OperationGetAll, "GET_ALL"},
		{"GetByID operation", OperationGetByID, "GET_BY_ID"},
		{"Update operation", OperationUpdate, "UPDATE"},
		{"Delete operation", OperationDelete, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.op) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.op))
			}
		})
	}
}

func TestNoOpHooks_StateProcessor(t *testing.T) {
	hooks := NoOpHooks[testModel]{}
	model := &testModel{ID: "1", Name: "Test"}

	operations := []Operation{
		OperationCreate,
		OperationGetAll,
		OperationGetByID,
		OperationUpdate,
		OperationDelete,
	}

	for _, op := range operations {
		t.Run(string(op), func(t *testing.T) {
			err := hooks.StateProcessor(context.Background(), op, "1", model)
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
		})
	}
}

func TestNoOpHooks_StateProcessor_WithNilModel(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	err := hooks.StateProcessor(context.Background(), OperationCreate, "1", nil)
	if err != nil {
		t.Errorf("Expected nil error with nil model, got %v", err)
	}
}

func TestNoOpHooks_StateProcessor_WithNilContext(t *testing.T) {
	hooks := NoOpHooks[testModel]{}
	model := &testModel{ID: "1", Name: "Test"}

	// Should not panic with nil context
	err := hooks.StateProcessor(nil, OperationCreate, "1", model)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestNoOpHooks_BeforeQuery(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	tests := []struct {
		name          string
		operation     Operation
		query         string
		args          []any
		expectedQuery string
		expectedArgs  []any
	}{
		{
			name:          "simple query",
			operation:     OperationGetAll,
			query:         "SELECT * FROM users",
			args:          []any{},
			expectedQuery: "SELECT * FROM users",
			expectedArgs:  []any{},
		},
		{
			name:          "query with args",
			operation:     OperationGetByID,
			query:         "SELECT * FROM users WHERE id = $1",
			args:          []any{"123"},
			expectedQuery: "SELECT * FROM users WHERE id = $1",
			expectedArgs:  []any{"123"},
		},
		{
			name:          "query with multiple args",
			operation:     OperationUpdate,
			query:         "UPDATE users SET name = $1 WHERE id = $2",
			args:          []any{"John", "123"},
			expectedQuery: "UPDATE users SET name = $1 WHERE id = $2",
			expectedArgs:  []any{"John", "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultQuery, resultArgs, err := hooks.BeforeQuery(
				context.Background(),
				tt.operation,
				tt.query,
				tt.args,
			)

			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}

			if resultQuery != tt.expectedQuery {
				t.Errorf("Expected query %s, got %s", tt.expectedQuery, resultQuery)
			}

			if len(resultArgs) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(resultArgs))
			}

			for i, expectedArg := range tt.expectedArgs {
				if resultArgs[i] != expectedArg {
					t.Errorf("Expected arg[%d]=%v, got %v", i, expectedArg, resultArgs[i])
				}
			}
		})
	}
}

func TestNoOpHooks_AfterQuery(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	tests := []struct {
		name      string
		operation Operation
		query     string
		args      []any
		result    any
		err       error
	}{
		{
			name:      "successful query",
			operation: OperationGetAll,
			query:     "SELECT * FROM users",
			args:      []any{},
			result:    []testModel{{ID: "1", Name: "Test"}},
			err:       nil,
		},
		{
			name:      "query with error",
			operation: OperationGetByID,
			query:     "SELECT * FROM users WHERE id = $1",
			args:      []any{"123"},
			result:    nil,
			err:       nil, // NoOpHooks doesn't propagate errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hooks.AfterQuery(
				context.Background(),
				tt.operation,
				tt.query,
				tt.args,
				tt.result,
				tt.err,
			)

			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
		})
	}
}

func TestNoOpHooks_ModifySelectQuery(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	// Create a simple select query builder
	// Note: We can't fully test the builder without a dialect, but we can verify the hook behavior
	builder := &query.SelectBuilder{}

	resultBuilder, modified := hooks.ModifySelectQuery(
		context.Background(),
		OperationGetAll,
		builder,
	)

	if modified {
		t.Error("Expected modified=false for NoOpHooks")
	}

	if resultBuilder != builder {
		t.Error("Expected same builder instance to be returned")
	}
}

func TestNoOpHooks_ModifyUpdateQuery(t *testing.T) {
	hooks := NoOpHooks[testModel]{}
	model := &testModel{ID: "1", Name: "Test"}

	builder := &query.UpdateBuilder{}

	resultBuilder, modified := hooks.ModifyUpdateQuery(
		context.Background(),
		OperationUpdate,
		"1",
		model,
		builder,
	)

	if modified {
		t.Error("Expected modified=false for NoOpHooks")
	}

	if resultBuilder != builder {
		t.Error("Expected same builder instance to be returned")
	}
}

func TestNoOpHooks_ModifyDeleteQuery(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	builder := &query.DeleteBuilder{}

	resultBuilder, modified := hooks.ModifyDeleteQuery(
		context.Background(),
		OperationDelete,
		"1",
		builder,
	)

	if modified {
		t.Error("Expected modified=false for NoOpHooks")
	}

	if resultBuilder != builder {
		t.Error("Expected same builder instance to be returned")
	}
}

func TestNoOpHooks_SerializeOne(t *testing.T) {
	hooks := NoOpHooks[testModel]{}
	model := &testModel{ID: "1", Name: "Test"}

	operations := []Operation{
		OperationCreate,
		OperationGetByID,
		OperationUpdate,
	}

	for _, op := range operations {
		t.Run(string(op), func(t *testing.T) {
			err := hooks.SerializeOne(context.Background(), op, model)
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
		})
	}
}

func TestNoOpHooks_SerializeOne_WithNilModel(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	err := hooks.SerializeOne(context.Background(), OperationGetByID, nil)
	if err != nil {
		t.Errorf("Expected nil error with nil model, got %v", err)
	}
}

func TestNoOpHooks_SerializeMany(t *testing.T) {
	hooks := NoOpHooks[testModel]{}

	tests := []struct {
		name   string
		models *[]testModel
	}{
		{
			name:   "empty slice",
			models: &[]testModel{},
		},
		{
			name: "single model",
			models: &[]testModel{
				{ID: "1", Name: "Test1"},
			},
		},
		{
			name: "multiple models",
			models: &[]testModel{
				{ID: "1", Name: "Test1"},
				{ID: "2", Name: "Test2"},
				{ID: "3", Name: "Test3"},
			},
		},
		{
			name:   "nil slice",
			models: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hooks.SerializeMany(context.Background(), OperationGetAll, tt.models)
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
		})
	}
}

// Test NoOpHooks with different generic types
func TestNoOpHooks_DifferentTypes(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		hooks := NoOpHooks[string]{}
		value := "test"
		err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("int type", func(t *testing.T) {
		hooks := NoOpHooks[int]{}
		value := 42
		err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("map type", func(t *testing.T) {
		hooks := NoOpHooks[map[string]interface{}]{}
		value := map[string]interface{}{"key": "value"}
		err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})
}

// Test that NoOpHooks implements the Hooks interface
func TestNoOpHooks_ImplementsHooksInterface(t *testing.T) {
	var _ Hooks[testModel] = NoOpHooks[testModel]{}
	var _ StateProcessor[testModel] = NoOpHooks[testModel]{}
	var _ SQLQueryListener[testModel] = NoOpHooks[testModel]{}
	var _ SQLQueryBuilderModifier[testModel] = NoOpHooks[testModel]{}
	var _ Serializer[testModel] = NoOpHooks[testModel]{}
}
