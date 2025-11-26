package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const UserContextKey contextKey = "authenticated_user"

const (
	UsersTable = "users"
)

type AuthenticatedUser struct {
	UserID    string
	Email     string
	Firstname string
	Lastname  string
}

type AuthPlugin struct {
	jwtSecret string
	db        database.Database
	jwtTTL    int
	app       *fiber.App
}

func NewPlugin() plugin.Plugin {
	return &AuthPlugin{}
}

func (p *AuthPlugin) Name() string {
	return "auth"
}

func (p *AuthPlugin) Initialize(config map[string]interface{}) error {
	if secret, ok := config["jwt_secret"].(string); ok {
		p.jwtSecret = secret
	}
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	}
	if ttl, ok := config["jwt_ttl"].(int); ok {
		p.jwtTTL = ttl
	}
	return nil
}

// Handler returns the auth middleware.
// This middleware checks for valid JWT tokens and sets the authenticated user in context.
// It does NOT automatically protect routes - you must explicitly apply it to protected routes.
func (p *AuthPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(p.jwtSecret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return c.Status(401).JSON(fiber.Map{"error": "token expired"})
			}
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		if !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			user := &AuthenticatedUser{}

			if userID, ok := claims["userId"].(string); ok {
				user.UserID = userID
			}
			if email, ok := claims["email"].(string); ok {
				user.Email = email
			}
			if firstname, ok := claims["firstname"].(string); ok {
				user.Firstname = firstname
			}
			if lastname, ok := claims["lastname"].(string); ok {
				user.Lastname = lastname
			}

			c.Locals(UserContextKey, user)
		}

		return c.Next()
	}
}

// SetupEndpoints implements the optional EndpointSetup interface
func (p *AuthPlugin) SetupEndpoints(app *fiber.App) error {
	if p.db == nil {
		return nil // Skip if no database configured
	}
	SetupAuth(app, p.db, p.jwtSecret, p.jwtTTL)
	return nil
}

func GetAuthenticatedUser(c *fiber.Ctx) *AuthenticatedUser {
	if user, ok := c.Locals(UserContextKey).(*AuthenticatedUser); ok {
		return user
	}
	return nil
}

func Context(c *fiber.Ctx) context.Context {
	ctx := c.Context()
	if user := GetAuthenticatedUser(c); user != nil {
		return context.WithValue(ctx, "user_id", user.UserID)
	}
	return ctx
}

func SetupAuth(app *fiber.App, db database.Database, jwtSecret string, jwtTTL int) {
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again in 15 minutes.",
			})
		},
	})

	app.Post("/login", loginLimiter, func(c *fiber.Ctx) error {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.Email == "" || body.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Email and password are required"})
		}

		var userId, storedPassword, firstname, lastname string

		query := `SELECT id, password, firstname, lastname FROM ` + UsersTable + ` WHERE email = ` + db.Dialect().Placeholder(1)
		err := db.QueryRow(context.Background(), query, body.Email).Scan(&userId, &storedPassword, &firstname, &lastname)

		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		if err := verifyPassword(body.Password, storedPassword); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		now := time.Now()
		claims := jwt.MapClaims{
			"userId":    userId,
			"email":     body.Email,
			"firstname": firstname,
			"lastname":  lastname,
			"iat":       now.Unix(),
			"exp":       now.Add(time.Duration(jwtTTL) * time.Second).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		t, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
		}

		return c.JSON(fiber.Map{
			"token": t,
			"user": fiber.Map{
				"id":        userId,
				"email":     body.Email,
				"firstname": firstname,
				"lastname":  lastname,
			},
		})
	})
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
