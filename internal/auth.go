package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nicolasbonnici/gorest/pkg/database"
)

func SetupAuth(app *fiber.App, db database.Database, jwtSecret string) {
	app.Post("/login", func(c *fiber.Ctx) error {
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

		passwordHash := hashPassword(body.Password, userId)
		if passwordHash != storedPassword {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		claims := jwt.MapClaims{
			"user_id":   userId,
			"email":     body.Email,
			"firstname": firstname,
			"lastname":  lastname,
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

func hashPassword(password, userId string) string {
	hash := sha256.Sum256([]byte("salt" + password + userId))
	return hex.EncodeToString(hash[:])
}
