package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/auth/handlers"
	"github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/auth/middleware"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
)

// Service manages authentication functionality
type Service struct {
	config config.AuthConfig
	jwt    *jwt.Service
	db     database.Database
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
		config: config,
		jwt:    jwt.NewService(config.JWTSecret, config.JWTTTL),
		db:     db,
	}, nil
}

// RegisterRoutes registers authentication endpoints
func (s *Service) RegisterRoutes(router fiber.Router) {
	handlers.RegisterAuthRoutes(router, s.db, s.jwt)
}

// Middleware returns the authentication middleware
func (s *Service) Middleware() fiber.Handler {
	return middleware.AuthMiddleware(s.jwt, s.db)
}

// OptionalMiddleware returns optional authentication middleware
func (s *Service) OptionalMiddleware() fiber.Handler {
	return middleware.OptionalAuthMiddleware(s.jwt, s.db)
}
