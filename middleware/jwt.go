package middleware

import (
	"yt-converter-api/config"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
)

func JWTProtected(c *fiber.Ctx) error {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(config.LoadConfig().JwtSecret)},
		ContextKey: "jwt",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Return status 401 and failed authentication error.
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No estás autorizado para acceder a este recurso",
			})
		},
	})(c)
}
