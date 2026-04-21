package rbac

import (
	"context"
	"reflect"
	"testing"
)

func TestWithRoles(t *testing.T) {
	ctx := context.Background()
	roles := []string{"admin", "user"}

	ctx = WithRoles(ctx, roles)

	value := ctx.Value(rolesContextKey)
	if value == nil {
		t.Fatal("expected roles to be in context")
	}

	retrievedRoles, ok := value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", value)
	}

	if !reflect.DeepEqual(retrievedRoles, roles) {
		t.Errorf("expected roles %v, got %v", roles, retrievedRoles)
	}
}

func TestGetRoles(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectedRoles []string
		expectedOk    bool
	}{
		{
			name: "roles exist in context",
			setupContext: func() context.Context {
				return WithRoles(context.Background(), []string{"admin", "user"})
			},
			expectedRoles: []string{"admin", "user"},
			expectedOk:    true,
		},
		{
			name: "no roles in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedRoles: nil,
			expectedOk:    false,
		},
		{
			name: "empty roles slice",
			setupContext: func() context.Context {
				return WithRoles(context.Background(), []string{})
			},
			expectedRoles: []string{},
			expectedOk:    true,
		},
		{
			name: "single role",
			setupContext: func() context.Context {
				return WithRoles(context.Background(), []string{"user"})
			},
			expectedRoles: []string{"user"},
			expectedOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			roles, ok := GetRoles(ctx)

			if ok != tt.expectedOk {
				t.Errorf("expected ok=%v, got %v", tt.expectedOk, ok)
			}

			if !reflect.DeepEqual(roles, tt.expectedRoles) {
				t.Errorf("expected roles %v, got %v", tt.expectedRoles, roles)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user123"

	ctx = WithUserID(ctx, userID)

	value := ctx.Value(userIDContextKey)
	if value == nil {
		t.Fatal("expected user ID to be in context")
	}

	retrievedID, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}

	if retrievedID != userID {
		t.Errorf("expected user ID %s, got %s", userID, retrievedID)
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectedID   string
		expectedOk   bool
	}{
		{
			name: "user ID exists in context",
			setupContext: func() context.Context {
				return WithUserID(context.Background(), "user123")
			},
			expectedID: "user123",
			expectedOk: true,
		},
		{
			name: "no user ID in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedID: "",
			expectedOk: false,
		},
		{
			name: "empty user ID",
			setupContext: func() context.Context {
				return WithUserID(context.Background(), "")
			},
			expectedID: "",
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			userID, ok := GetUserID(ctx)

			if ok != tt.expectedOk {
				t.Errorf("expected ok=%v, got %v", tt.expectedOk, ok)
			}

			if userID != tt.expectedID {
				t.Errorf("expected user ID %s, got %s", tt.expectedID, userID)
			}
		})
	}
}

func TestWithUser(t *testing.T) {
	ctx := context.Background()
	userID := "user123"
	roles := []string{"admin", "editor"}

	ctx = WithUser(ctx, userID, roles)

	retrievedID, ok := GetUserID(ctx)
	if !ok {
		t.Error("expected user ID to be in context")
	}
	if retrievedID != userID {
		t.Errorf("expected user ID %s, got %s", userID, retrievedID)
	}

	retrievedRoles, ok := GetRoles(ctx)
	if !ok {
		t.Error("expected roles to be in context")
	}
	if !reflect.DeepEqual(retrievedRoles, roles) {
		t.Errorf("expected roles %v, got %v", roles, retrievedRoles)
	}
}

func TestContextKeyUniqueness(t *testing.T) {
	if rolesContextKey == userIDContextKey {
		t.Error("context keys should be unique")
	}

	ctx := context.Background()
	ctx = WithRoles(ctx, []string{"admin"})
	ctx = WithUserID(ctx, "user123")

	roles, ok := GetRoles(ctx)
	if !ok || len(roles) != 1 || roles[0] != "admin" {
		t.Error("roles should be preserved independently")
	}

	userID, ok := GetUserID(ctx)
	if !ok || userID != "user123" {
		t.Error("user ID should be preserved independently")
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	ctx = WithRoles(ctx, []string{"user"})

	ctx = WithUserID(ctx, "user123")

	ctx = WithRoles(ctx, []string{"admin", "editor"})

	roles, ok := GetRoles(ctx)
	if !ok {
		t.Fatal("expected roles in context")
	}

	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}

	userID, ok := GetUserID(ctx)
	if !ok {
		t.Fatal("expected user ID in context")
	}

	if userID != "user123" {
		t.Errorf("expected user ID 'user123', got %s", userID)
	}
}

func TestContextImmutability(t *testing.T) {
	baseCtx := context.Background()

	ctx1 := WithRoles(baseCtx, []string{"admin"})
	ctx2 := WithRoles(baseCtx, []string{"user"})

	roles1, ok1 := GetRoles(ctx1)
	roles2, ok2 := GetRoles(ctx2)

	if !ok1 || !ok2 {
		t.Fatal("expected roles in both contexts")
	}

	if len(roles1) != 1 || roles1[0] != "admin" {
		t.Error("ctx1 should have admin role")
	}

	if len(roles2) != 1 || roles2[0] != "user" {
		t.Error("ctx2 should have user role")
	}

	_, ok := GetRoles(baseCtx)
	if ok {
		t.Error("base context should not have roles")
	}
}

func TestContextValueTypes(t *testing.T) {
	ctx := context.Background()

	ctx = context.WithValue(ctx, rolesContextKey, "not a slice")

	_, ok := GetRoles(ctx)
	if ok {
		t.Error("GetRoles should return false for non-[]string value")
	}

	ctx = context.WithValue(ctx, userIDContextKey, 123)

	_, ok = GetUserID(ctx)
	if ok {
		t.Error("GetUserID should return false for non-string value")
	}
}
