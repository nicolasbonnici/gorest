package context

import "github.com/gofiber/fiber/v2"

const userIDKey = "user_id"

func SetUserID(c *fiber.Ctx, userID string) {
	c.Locals(userIDKey, userID)
}

func GetUserID(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals(userIDKey).(string)
	return userID, ok
}

func MustGetUserID(c *fiber.Ctx) string {
	userID, _ := GetUserID(c)
	return userID
}
