package handlers

import (
	stdcontext "context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/auth/models"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/response"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func RegisterAuthRoutes(router fiber.Router, db database.Database, jwtService *jwt.Service) {
	authGroup := router.Group("/auth")
	userCRUD := crud.New[models.User](db)

	authGroup.Post("/register", handleRegister(db, userCRUD, jwtService))
	authGroup.Post("/login", handleLogin(db, jwtService))
	authGroup.Post("/refresh", handleRefresh(jwtService))
}

func handleRegister(db database.Database, userCRUD *crud.CRUD[models.User], jwtService *jwt.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		if err := checkEmailExists(ctx, db, req.Email, uuid.Nil); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}

		password := req.Password
		user := models.User{
			ID:        uuid.New(),
			Email:     req.Email,
			Password:  &password,
			Firstname: req.Firstname,
			Lastname:  req.Lastname,
			Role:      "user",
			CreatedAt: time.Now(),
		}

		if err := user.HashPassword(); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to hash password")
		}

		if err := userCRUD.Create(ctx, user); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to create user")
		}

		token, err := jwtService.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendCreated(c, AuthResponse{
			Token: token,
			User:  &user,
		})
	}
}

func handleLogin(db database.Database, jwtService *jwt.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		user, err := getUserByEmail(ctx, db, req.Email)
		if err != nil {
			return err
		}

		if !user.CheckPassword(req.Password) {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}

		token, err := jwtService.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendFormatted(c, fiber.StatusOK, AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

func handleRefresh(jwtService *jwt.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type RefreshRequest struct {
			Token string `json:"token" validate:"required"`
		}

		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		newToken, err := jwtService.RefreshToken(req.Token)
		if err != nil {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		return response.SendFormatted(c, fiber.StatusOK, fiber.Map{
			"token": newToken,
		})
	}
}

func checkEmailExists(ctx stdcontext.Context, db database.Database, email string, excludeUserID uuid.UUID) error {
	qb := query.New(db.Dialect()).
		Select("email").
		From("users").
		Where(query.Eq("email", email))

	if excludeUserID != uuid.Nil {
		qb = qb.Where(query.Ne("id", excludeUserID))
	}

	queryStr, args, err := qb.Build()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	var existingEmail string
	err = db.QueryRow(ctx, queryStr, args...).Scan(&existingEmail)
	if err == nil {
		if excludeUserID == uuid.Nil {
			return fmt.Errorf("user with this email already exists")
		}
		return fmt.Errorf("email already in use")
	}
	if !crud.IsNotFoundError(err) {
		return fmt.Errorf("failed to check existing email: %w", err)
	}

	return nil
}

func getUserByEmail(ctx stdcontext.Context, db database.Database, email string) (*models.User, error) {
	qb := query.New(db.Dialect()).
		Select("id", "firstname", "lastname", "email", "password", "role", "created_at", "updated_at").
		From("users").
		Where(query.Eq("email", email))

	queryStr, args, err := qb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var user models.User
	var password *string
	var updatedAt *time.Time
	err = db.QueryRow(ctx, queryStr, args...).
		Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email, &password, &user.Role, &user.CreatedAt, &updatedAt)
	if crud.IsNotFoundError(err) {
		return nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user.Password = password
	user.UpdatedAt = updatedAt

	return &user, nil
}
