package handlers

import (
	stdcontext "context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/auth/converters"
	"github.com/nicolasbonnici/gorest/auth/dtos"
	"github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/auth/models"
	"github.com/nicolasbonnici/gorest/auth/refresh"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/response"
	"golang.org/x/crypto/bcrypt"
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
	Token        string                `json:"token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresIn    int                   `json:"expires_in"`
	User         *dtos.UserResponseDTO `json:"user"`
}

type TokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// RegisterAuthRoutes mounts the credential endpoints. throttle is applied to
// every route that accepts or exchanges a secret; pass nil to mount them
// unthrottled (the tests do, so a table of cases is not fighting a limiter).
func RegisterAuthRoutes(router fiber.Router, db database.Database, jwtService *jwt.Service, refreshService *refresh.Service, throttle fiber.Handler) {
	authGroup := router.Group("/auth")
	userCRUD := crud.New[models.User](db)

	register := handleRegister(db, userCRUD, jwtService, refreshService)
	login := handleLogin(db, jwtService, refreshService)
	refreshHandler := handleRefresh(jwtService, refreshService)

	if throttle != nil {
		// Logout only invalidates a token the caller already holds, so it is
		// left off the throttle: rate-limiting it would push a client that is
		// trying to end its session into keeping it alive.
		authGroup.Post("/register", throttle, register)
		authGroup.Post("/login", throttle, login)
		authGroup.Post("/refresh", throttle, refreshHandler)
	} else {
		authGroup.Post("/register", register)
		authGroup.Post("/login", login)
		authGroup.Post("/refresh", refreshHandler)
	}
	authGroup.Post("/logout", handleLogout(refreshService))
}

func handleRegister(db database.Database, userCRUD *crud.CRUD[models.User], jwtService *jwt.Service, refreshService *refresh.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req RegisterRequest
		if err := c.Bind().Body(&req); err != nil {
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

		refreshToken, err := refreshService.Issue(ctx, user.ID)
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to issue refresh token")
		}

		roles, _ := user.GetRoles(ctx, db)

		converter := &converters.UserConverter{}
		userDTO := converter.ModelToResponseDTO(user)
		userDTO.Roles = roles

		return response.SendCreated(c, AuthResponse{
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresIn:    jwtService.TTL(),
			User:         &userDTO,
		})
	}
}

func handleLogin(db database.Database, jwtService *jwt.Service, refreshService *refresh.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().Body(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		user, err := getUserByEmail(ctx, db, req.Email)
		if errors.Is(err, errUserNotFound) {
			// Burn a bcrypt comparison anyway. Skipping it would answer an
			// unknown address in a fraction of the time a known one takes,
			// which enumerates accounts just as well as a distinct status code.
			equalizeLoginTiming(req.Password)
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "authentication failed")
		}

		if !user.CheckPassword(req.Password) {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}

		token, err := jwtService.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		refreshToken, err := refreshService.Issue(ctx, user.ID)
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to issue refresh token")
		}

		roles, _ := user.GetRoles(ctx, db)

		converter := &converters.UserConverter{}
		userDTO := converter.ModelToResponseDTO(*user)
		userDTO.Roles = roles

		return response.SendFormatted(c, fiber.StatusOK, AuthResponse{
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresIn:    jwtService.TTL(),
			User:         &userDTO,
		})
	}
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func handleRefresh(jwtService *jwt.Service, refreshService *refresh.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req RefreshRequest
		if err := c.Bind().Body(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		rotated, newRefreshToken, err := refreshService.Rotate(ctx, req.RefreshToken)
		if err != nil {
			switch {
			case errors.Is(err, refresh.ErrTokenReuse):
				// The family is already revoked at this point; say so explicitly
				// so clients stop retrying and force a fresh login.
				return response.SendError(c, fiber.StatusUnauthorized, "refresh token reuse detected, all sessions revoked")
			case errors.Is(err, refresh.ErrInvalidToken), errors.Is(err, refresh.ErrExpiredToken):
				return response.SendError(c, fiber.StatusUnauthorized, "invalid or expired refresh token")
			default:
				return response.SendError(c, fiber.StatusInternalServerError, "failed to refresh token")
			}
		}

		token, err := jwtService.GenerateToken(rotated.UserID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendFormatted(c, fiber.StatusOK, TokenResponse{
			Token:        token,
			RefreshToken: newRefreshToken,
			ExpiresIn:    jwtService.TTL(),
		})
	}
}

func handleLogout(refreshService *refresh.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req RefreshRequest
		if err := c.Bind().Body(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		if err := refreshService.Revoke(c.Context(), req.RefreshToken); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to revoke refresh token")
		}

		return c.SendStatus(fiber.StatusNoContent)
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

// errUserNotFound separates "no such account" from a genuine lookup failure.
// The caller must answer both with the same status and message, or the login
// endpoint tells an attacker which addresses are registered.
var errUserNotFound = errors.New("user not found")

// dummyHash is a valid bcrypt digest of a value nothing can log in with. It
// exists solely to give the unknown-account path the same cost as the real one.
var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func equalizeLoginTiming(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

func getUserByEmail(ctx stdcontext.Context, db database.Database, email string) (*models.User, error) {
	qb := query.New(db.Dialect()).
		Select("id", "firstname", "lastname", "email", "password", "created_at", "updated_at").
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
		Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email, &password, &user.CreatedAt, &updatedAt)
	if crud.IsNotFoundError(err) {
		return nil, errUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user.Password = password
	user.UpdatedAt = updatedAt

	return &user, nil
}
