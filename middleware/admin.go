package middleware

import (
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func IsAdmin(c *fiber.Ctx) error {
	jwt := c.Locals("jwt").(*jwt.Token)

	// Obtener el usuario del token
	_, role, err := pkg.GetUserFromToken(jwt.Raw)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No tienes permisos para acceder a esta ruta",
		})
	}

	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo los administradores pueden acceder a esta ruta",
		})
	}

	return c.Next()
}
