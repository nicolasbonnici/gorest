package internal

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func SetupAuth(app *fiber.App, jwtSecret string) {
	app.Post("/login", func(c *fiber.Ctx) error {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(err.Error())
		}

		if body.Username == "admin" && body.Password == "password" {
			claims := jwt.MapClaims{
				"username": body.Username,
				"role":     "admin",
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			t, _ := token.SignedString([]byte(jwtSecret))
			return c.JSON(fiber.Map{"token": t})
		}
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	})
}

func JWTMiddleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}
		return c.Next()
	}
}
