package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/auth/handlers"
	"github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/auth/middleware"
	"github.com/nicolasbonnici/gorest/auth/refresh"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/logger"
)

// Service manages authentication functionality
type Service struct {
	config  config.AuthConfig
	jwt     *jwt.Service
	refresh *refresh.Service
	db      database.Database
}

// NewService creates a new auth service
func NewService(authConfig config.AuthConfig, db database.Database) (*Service, error) {
	config := authConfig
	if !config.Enabled {
		return nil, nil
	}

	if config.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret is required when auth is enabled")
	}

	if config.JWTTTL == 0 {
		config.JWTTTL = 900 // Default 15 minutes
	}

	return &Service{
		config:  config,
		jwt:     jwt.NewService(config.JWTSecret, config.JWTTTL),
		refresh: refresh.NewService(db, config.RefreshTTL),
		db:      db,
	}, nil
}

// RegisterRoutes registers authentication endpoints
func (s *Service) RegisterRoutes(router fiber.Router) {
	handlers.RegisterAuthRoutes(router, s.db, s.jwt, s.refresh)
}

// RefreshService exposes the refresh token store so applications can revoke
// sessions (for example on password change) or run cleanup on their own schedule.
func (s *Service) RefreshService() *refresh.Service {
	return s.refresh
}

// StartRefreshTokenCleanup periodically deletes expired refresh tokens until ctx
// is cancelled. Without it the table grows without bound, since expired rows are
// never removed on the request path.
func (s *Service) StartRefreshTokenCleanup(ctx context.Context) {
	// Expired tokens are already rejected on use, so sweeping is about disk
	// rather than correctness — hourly is frequent enough.
	const interval = time.Hour

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deleted, err := s.refresh.DeleteExpired(ctx, time.Now())
				if err != nil {
					logger.Log.Warn("Failed to clean up expired refresh tokens", "error", err)
					continue
				}
				if deleted > 0 {
					logger.Log.Info("Cleaned up expired refresh tokens", "deleted", deleted)
				}
			}
		}
	}()
}

// Middleware returns the authentication middleware
func (s *Service) Middleware() fiber.Handler {
	return middleware.AuthMiddleware(s.jwt, s.db)
}

// OptionalMiddleware returns optional authentication middleware
func (s *Service) OptionalMiddleware() fiber.Handler {
	return middleware.OptionalAuthMiddleware(s.jwt, s.db)
}
