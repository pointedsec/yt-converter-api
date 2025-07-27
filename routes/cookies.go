package routes

import (
	"fmt"
	"os"
	"path/filepath"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
)

// Subir archivo de Cookies para usar con yt-dlp
func UploadCookies(c *fiber.Ctx) error {
	// Obtener el archivo del formulario
	file, err := c.FormFile("cookies")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se pudo leer el archivo cookies",
		})
	}

	// Validar extensión .txt
	if filepath.Ext(file.Filename) != ".txt" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El archivo debe tener extensión .txt",
		})
	}

	// Validar tamaño (mínimo 0.5 KB, máximo 2 MB)
	const minSize int64 = 512       // 0.5 KB
	const maxSize int64 = 1024 * 2048 // 2 MB
	if file.Size < minSize || file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("El archivo debe pesar entre %d bytes y %d bytes", minSize, maxSize),
		})
	}

	// Validar Content-Type (opcional, según navegador puede ser inconsistente)
	header := file.Header.Get("Content-Type")
	if header != "" && header != "text/plain" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El archivo debe ser de tipo text/plain",
		})
	}

	// Guardar el archivo en una ruta fija
	savePath := "./pkg/pyConverter/cookies.txt"
	err = c.SaveFile(file, savePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al guardar el archivo cookies",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cookies Uploaded Succesfully",
	})
}

// Comprueba si el archivo de Cookies ya está subido para yt-dlp
func GetCookiesInfo(c *fiber.Ctx) error {
	info, err := pkg.CheckCookiesFile()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(info)
}

// Borra el archivo cookies.txt si existe.
func DeleteCookiesFile(c *fiber.Ctx) error {
	path := "./pkg/pyConverter/cookies.txt"

	// Verifica si el archivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "El archivo de cookies no existe",
		})
	}

	// Elimina el archivo
	err := os.Remove(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No se pudo eliminar el archivo de cookies",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Archivo de cookies eliminado correctamente",
	})
}
