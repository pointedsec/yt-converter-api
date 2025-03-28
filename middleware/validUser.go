package middleware

import (
	"net/http"
	"yt-converter-api/db"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Middleware para comprobar que el usuario existe (puede que un administrador haya borrado el usuario) y que además esté activo
func ValidUserAndActive(c *fiber.Ctx) error {
	jwt := c.Locals("jwt").(*jwt.Token)

	// Obtener el usuario del token
	userID, _, err := pkg.GetUserFromToken(jwt.Raw)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No tienes permisos para acceder a esta ruta",
		})
	}

	userExists := false
	err = db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ? AND active = 1", userID).Scan(&userExists)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el usuario existe",
		})
	}

	if !userExists {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "El usuario actual ha sido desactivado por un administrador",
		})
	}

	return c.Next()
}
