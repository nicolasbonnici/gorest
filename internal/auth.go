package internal

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupAuth(app *fiber.App, db *pgxpool.Pool, jwtSecret string) {
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

		var userId string
		var firstname, lastname string

		err := db.QueryRow(context.Background(),
			`SELECT id, firstname, lastname
			FROM `+UsersTable+`
			WHERE email = $1
			AND password = encode(digest('salt' || $2 || id::text, 'sha256'), 'hex')`,
			body.Email,
			body.Password,
		).Scan(&userId, &firstname, &lastname)

		if err != nil {
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
