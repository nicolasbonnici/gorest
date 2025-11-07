package internal

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nicolasbonnici/gorest/database"
	"golang.org/x/crypto/bcrypt"
)

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
