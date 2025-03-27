package routes

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"yt-converter-api/config"
	"yt-converter-api/db"
	"yt-converter-api/models"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// GetVideos obtiene la lista de videos íntegra
func GetVideos(c *fiber.Ctx) error {
	rows, err := db.DB.Query("SELECT * FROM videos")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
		}
		videos = append(videos, video)
	}

	return c.JSON(videos)
}

// AddVideo agrega un nuevo video si no existe
func AddVideo(c *fiber.Ctx) error {
	type Request struct {
		URL string `json:"url"`
	}

	var request Request
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}

	// Obtener el token JWT del contexto
	token := c.Locals("jwt").(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	userID := claims["user_id"].(string)

	// Convertir el userID de string a int
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al procesar el ID del usuario",
		})
	}

	// Comprobar la validez de la URL
	if !pkg.IsUrl(request.URL) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL no válida",
		})
	}

	// Comprobar si la URL es un video de Youtube
	valid, err := pkg.IsYoutubeUrl(request.URL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "La URL no es una URL válida de Youtube",
		})
	}

	if !valid {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL no es un video de Youtube",
		})
	}

	// Obtener el ID del video y el título
	videoTitle, err := pkg.GetYoutubeVideoTitle(request.URL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el título del video",
		})
	}

	// Insertar el video en la base de datos
	video := models.Video{
		UserID:        userIDInt,
		VideoID:       pkg.GetYoutubeVideoID(request.URL),
		Title:         videoTitle,
		RequestedByIP: c.IP(),
	}

	// Comprobar si el video ya existe en la base de datos
	var exists int
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM videos WHERE video_id = ?)", video.VideoID).Scan(&exists) // exists ya está definido en el bucle de la función AddVideo
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al verificar si el video existe",
		})
	}
	if exists == 1 {
		_, err = db.DB.Exec("UPDATE videos SET updated_at = CURRENT_TIMESTAMP WHERE video_id = ?", pkg.GetYoutubeVideoID(request.URL))
		msg := ""
		if err != nil {
			msg = "Ademas ha ocurrido un error al intentar actualizar la fecha actual del video que se quería agregar"
		}
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error":     "El video ya existe en la base de datos",
			"videoID":   video.VideoID,
			"extraInfo": msg,
		})
	}

	_, err = db.DB.Exec("INSERT INTO videos (user_id, video_id, title, requested_by_ip) VALUES (?, ?, ?, ?)", video.UserID, video.VideoID, video.Title, video.RequestedByIP)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al insertar el video",
			"errorTrace": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Video agregado correctamente",
		"videoID": video.VideoID,
	})
}

// GetVideo Obtiene un video en base del video_id
func GetVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	rows, err := db.DB.Query("SELECT * FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	var video models.Video
	rows.Next()
	err = rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
	}

	return c.JSON(video)
}

// DeleteVideo Elimina un video de la base de datos
func DeleteVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	tx, _ := db.DB.Begin()
	// Get all video_status for the video, there can be multiple
	rows, err := tx.Query("SELECT * FROM video_status WHERE video_id = ?", videoID)
	if err != nil {
		tx.Rollback()
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los video_status",
		})
	}
	defer rows.Close()

	var processed_videos []models.VideoStatus
	for rows.Next() {
		var status models.VideoStatus
		err = rows.Scan(&status.ID, &status.VideoID, &status.Resolution, &status.Path, &status.Status, &status.CreatedAt, &status.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
		}
		processed_videos = append(processed_videos, status)
	}

	// Delete processed videos
	_, err = tx.Exec("DELETE FROM video_status WHERE video_id = ?", videoID)
	if err != nil {
		tx.Rollback()
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al eliminar el video",
			"errorTrace": err.Error(),
		})
	}
	// Delete video
	_, err = tx.Exec("DELETE FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		tx.Rollback()
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al eliminar el video",
			"errorTrace": err.Error(),
		})
	}

	// Borrar videos procesados de este usuario
	if len(processed_videos) != 0 {
		for _, video := range processed_videos {
			if _, err := os.Stat(video.Path); err == nil {
				err := os.Remove(video.Path)
				if err != nil {
					tx.Rollback()
					return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
						"error": "Uno de los videos convertidos no se ha encontrado, comprobar manualmente las rutas en la base de datos y si el archivo realmente existe",
					})
				} else {
					fmt.Printf("✅ Borrado: %s\n", video.Path)
				}
			} else {
				tx.Rollback()
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "Uno de los videos convertidos de no se ha encontrado, comprobar manualmente las rutas en la base de datos y si el archivo realmente existe",
				})
			}
		}
	}
	tx.Commit()
	return c.JSON(fiber.Map{
		"message": "Video eliminado correctamente",
	})
}

// Obtiene las resoluciones disponibles para un video
func GetVideoFormats(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	rows, err := db.DB.Query("SELECT * FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	// Comprueba si el video ya está insertado en la base de datos
	var video models.Video
	rows.Next()
	err = rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
	}

	if video.ID == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Video no encontrado",
		})
	}

	resolutions, err := pkg.GetYoutubeVideoResolutions(video.VideoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resolutions)
}

// Procesa un video de forma asíncrona para obtener el formato indicado por POST
func ProcessVideo(c *fiber.Ctx) error {
	// Obtiene el formato por POST (json)
	type Request struct {
		Resolution string `json:"resolution"`
	}

	var request Request
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}

	resolution := request.Resolution

	// Obtiene el video_id por GET
	videoID := c.Params("video_id")

	// Comprobar si el video ya existe en la base de datos
	exists, err := db.DB.Query("SELECT COUNT(*) FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el video existe en la base de datos",
		})
	}
	defer exists.Close()

	var count int
	exists.Next()
	err = exists.Scan(&count)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el resultado de la consulta",
		})
	}

	if count == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "El video no existe en la base de datos",
		})
	}

	// Procesar el video de forma asíncrona
	go func() {
		if _, err := ProcessYoutubeVideo(videoID, resolution); err != nil {
			fmt.Printf("Error procesando video: %v\n", err)
		}
		_, err := db.DB.Exec("UPDATE videos SET updated_at = CURRENT_TIMESTAMP WHERE video_id = ?", pkg.GetYoutubeVideoID(videoID))
		if err != nil {
			fmt.Printf("Error actualizando el video para la fecha de modificación")
		}
	}()

	return c.JSON(fiber.Map{
		"message": "Video procesado, puedes consultar el estado en el endpoint GET /videos/:video_id/status",
	})
}

// Procesa un video de Youtube de forma asíncrona
func ProcessYoutubeVideo(videoID string, resolution string) (string, error) {
	// Comprobar si la resolución está disponible
	resolutions, err := pkg.GetYoutubeVideoResolutions(videoID)
	if err != nil {
		return "", fmt.Errorf("error al obtener las resoluciones del video: %v", err)
	}

	if !slices.Contains(resolutions, resolution) {
		return "", fmt.Errorf("la resolución %s no está disponible", resolution)
	}

	// Insertar el video procesandose en la base de datos
	_, err = db.DB.Exec("INSERT INTO video_status (video_id, resolution, status) VALUES (?, ?, ?)", videoID, resolution, "processing")
	if err != nil {
		return "", fmt.Errorf("error al insertar el estado del video: %v", err)
	}

	// Ejecutar el comando para descargar el video haciendo uso de pyConverter/main.py
	cmd := exec.Command("/usr/bin/python3", config.LoadConfig().PyConverterPath, videoID, "video", config.LoadConfig().StoragePath, resolution)

	output, err := cmd.Output()
	if err != nil {
		// Actualizar el estado del video en la base de datos
		_, updateErr := db.DB.Exec("UPDATE video_status SET status = ? WHERE video_id = ? AND resolution = ?", "failed", videoID, resolution)
		if updateErr != nil {
			fmt.Printf("Error actualizando estado a failed: %v\n", updateErr)
		}
		return "", fmt.Errorf("error al ejecutar el comando para descargar el video: %v, output: %v", err, string(output))
	}

	// Obtener la última línea del output (que contiene el path)
	outputLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(outputLines) == 0 {
		return "", fmt.Errorf("no se obtuvo output del comando")
	}
	videoPath := outputLines[len(outputLines)-1]

	// Actualizar el estado del video y el path en la base de datos
	_, err = db.DB.Exec("UPDATE video_status SET status = ?, path = ? WHERE video_id = ? AND resolution = ?", "completed", videoPath, videoID, resolution)
	if err != nil {
		return "", fmt.Errorf("error al actualizar el estado del video: %v", err)
	}

	return videoPath, nil
}

// Obtiene el estado del video, pueden haber varias resoluciones por video
func GetVideoStatus(c *fiber.Ctx) error {
	videoID := c.Params("video_id")

	// Primero verificamos si el video existe
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM video_status WHERE video_id = ?)", videoID).Scan(&exists)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al verificar si existe el estado del video",
		})
	}

	if !exists {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "No se encontró ningún estado para este video",
		})
	}

	// Si existe, obtenemos todos los estados
	rows, err := db.DB.Query("SELECT id, video_id, resolution, path, status, created_at, updated_at FROM video_status WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el estado del video",
		})
	}
	defer rows.Close()

	var videoStatus []models.VideoStatus
	for rows.Next() {
		var status models.VideoStatus
		var path *string // Usamos un puntero para manejar NULL
		err = rows.Scan(
			&status.ID,
			&status.VideoID,
			&status.Resolution,
			&path,
			&status.Status,
			&status.CreatedAt,
			&status.UpdatedAt,
		)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":      "Error al leer el estado del video",
				"errorTrace": err.Error(),
			})
		}
		// Si path no es NULL, asignamos su valor
		if path != nil {
			status.Path = *path
		}
		videoStatus = append(videoStatus, status)
	}

	if err = rows.Err(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al procesar los resultados",
		})
	}

	return c.JSON(videoStatus)
}

// Descarga un video que ya ha sido procesado
func DownloadVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	// Get resolution from Query Params
	resolution := c.Query("resolution")

	// Comprobar si el video procesado ya existe en la base de datos
	exists, err := db.DB.Query("SELECT COUNT(*) FROM video_status WHERE video_id = ? AND resolution = ?", videoID, resolution)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el video procesado existe en la base de datos",
		})
	}
	defer exists.Close()

	var count int
	exists.Next()
	err = exists.Scan(&count)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el resultado de la consulta",
		})
	}

	if count == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "El video procesado no existe en la base de datos",
		})
	}

	// Obtener el path del video procesado
	var path string
	err = db.DB.QueryRow("SELECT path FROM video_status WHERE video_id = ? AND resolution = ?", videoID, resolution).Scan(&path)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el path del video procesado",
		})
	}

	// Comprobar si el path existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Borrar el estado del video en la base de datos
		_, err = db.DB.Exec("DELETE FROM video_status WHERE video_id = ? AND resolution = ?", videoID, resolution)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al borrar el estado del video",
			})
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "El video procesado no existe en la base de datos",
		})
	}

	// Get video title name
	var title string
	err = db.DB.QueryRow("SELECT title FROM videos WHERE video_id = ?", videoID).Scan(&title)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el título del video procesado",
		})
	}

	// Descargar el video
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+title+`"`)
	return c.SendFile(path)
}
