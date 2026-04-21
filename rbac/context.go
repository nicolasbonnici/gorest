package rbac

import (
	"context"
)

type contextKey string

const (
	rolesContextKey  contextKey = "rbac:roles"
	userIDContextKey contextKey = "rbac:user_id"
	fiberContextKey  contextKey = "rbac:fiber_ctx"
)

func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, rolesContextKey, roles)
}

func GetRoles(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(rolesContextKey).([]string)
	return roles, ok
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

func WithUser(ctx context.Context, userID string, roles []string) context.Context {
	ctx = WithUserID(ctx, userID)
	ctx = WithRoles(ctx, roles)
	return ctx
}
