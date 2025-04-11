package routes
import (
	"github.com/gofiber/fiber/v2"
)

func GetStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"active": true,
	})
}