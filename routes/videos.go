package routes

import (
	"database/sql"
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
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM videos WHERE video_id = ?)", video.VideoID).Scan(&exists)
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
		return c.JSON(fiber.Map{
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
						"error": "Uno de los videos convertidos no se ha podido borrar, comprobar manualmente las rutas en la base de datos y si el archivo realmente existe",
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

	var cookiesPath string

	// Comprobar si la petición es multipart/form-data antes de intentar leer archivo
	contentType := c.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		fileHeader, err := c.FormFile("cookies")
		if err != nil {
			// Si el error indica que no hay archivo, continuar sin cookies
			if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "http: no such file") {
				fileHeader = nil
			} else {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error leyendo archivo cookies: " + err.Error(),
				})
			}
		}

		if fileHeader != nil {
			tempFile, err := os.CreateTemp("", fmt.Sprintf("cookies_%s_*.txt", videoID))
			if err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "No se pudo crear archivo temporal para cookies",
				})
			}
			cookiesPath = tempFile.Name()
			tempFile.Close()

			err = c.SaveFile(fileHeader, cookiesPath)
			if err != nil {
				os.Remove(cookiesPath)
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "No se pudo guardar el archivo cookies.txt",
				})
			}
			defer os.Remove(cookiesPath)
		}
	}

	// Verificar existencia del video
	row := db.DB.QueryRow("SELECT id, user_id, video_id, title, requested_by_ip, created_at, updated_at FROM videos WHERE video_id = ?", videoID)

	var video models.Video
	err := row.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Video no encontrado",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el video",
		})
	}

	// Obtener resoluciones pasando el path del archivo cookies si existe (vacío si no)
	resolutions, err := pkg.GetYoutubeVideoResolutions(video.VideoID, cookiesPath)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resolutions)
}

// Procesa un video de forma asíncrona obteniendo la resolución indicada por POST
func ProcessVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")

	// Leer los campos de texto del formulario
	resolution := c.FormValue("Resolution", "720p") // valor por defecto
	isAudio := c.FormValue("IsAudio", "false") == "true"

	// Obtener el archivo cookies.txt (si existe)
	fileHeader, err := c.FormFile("cookies")
	var cookiesPath string

	if err == nil && fileHeader != nil {
		// Crear archivo temporal para guardar las cookies
		tempFile, err := os.CreateTemp("", fmt.Sprintf("cookies_%s_*.txt", videoID))
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "No se pudo crear archivo temporal para cookies",
			})
		}
		cookiesPath = tempFile.Name()
		tempFile.Close() // cerramos el archivo temporal para que SaveFile pueda escribirlo

		// Guardar el archivo cookies en el archivo temporal
		err = c.SaveFile(fileHeader, cookiesPath)
		if err != nil {
			os.Remove(cookiesPath) // intentar limpiar
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "No se pudo guardar el archivo cookies.txt",
			})
		}
	}

	// Verificar existencia del video
	row := db.DB.QueryRow("SELECT COUNT(*) FROM videos WHERE video_id = ?", videoID)
	var count int
	if err := row.Scan(&count); err != nil || count == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "El video no existe en la base de datos",
		})
	}

	// Procesar el video en segundo plano
	go func() {
		// Limpiar archivo después de usarlo
		defer os.Remove(cookiesPath)
		if _, err := ProcessYoutubeVideo(videoID, resolution, isAudio, cookiesPath); err != nil {
			fmt.Printf("Error procesando video: %v\n", err)
		}
		_, err := db.DB.Exec("UPDATE videos SET updated_at = CURRENT_TIMESTAMP WHERE video_id = ?", videoID)
		if err != nil {
			fmt.Printf("Error actualizando el video: %v\n", err)
		}
	}()

	return c.JSON(fiber.Map{
		"message": "Procesamiento iniciado",
	})
}

// Procesa un video de Youtube de forma asíncrona
func ProcessYoutubeVideo(videoID string, resolution string, isAudio bool, cookiesPath string) (string, error) {
	// Comprobar si la resolución está disponible solo si se va a descargar video
	if !isAudio {
		resolutions, err := pkg.GetYoutubeVideoResolutions(videoID, cookiesPath)
		if err != nil {
			return "", fmt.Errorf("error al obtener las resoluciones del video: %v", err)
		}
		if !slices.Contains(resolutions, resolution) {
			return "", fmt.Errorf("la resolución %s no está disponible", resolution)
		}
	} else {
		resolution = "mp3"
	}

	// Comprobar si ya está procesado con esa resolución
	var exists int
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM video_status WHERE video_id = ? AND resolution = ? and status = ?)", videoID, resolution, "completed").Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("error al verificar si el video ya está procesado: %v", err)
	}
	if exists == 1 {
		_, _ = db.DB.Exec("UPDATE video_status SET updated_at = CURRENT_TIMESTAMP WHERE video_id = ? and resolution = ? and status = ?", videoID, resolution, "completed")
		return "", fmt.Errorf("el video con id %v y resolución %v ya está procesado", videoID, resolution)
	}

	// Borrar estado fallido previo
	_, _ = db.DB.Exec("DELETE FROM video_status WHERE video_id = ? AND resolution = ? and status = ?", videoID, resolution, "failed")

	// Insertar nuevo estado: procesando
	_, err = db.DB.Exec("INSERT INTO video_status (video_id, resolution, status) VALUES (?, ?, ?)", videoID, resolution, "processing")
	if err != nil {
		return "", fmt.Errorf("error al insertar el estado del video: %v", err)
	}

	// Construir comando dinámico
	args := []string{
		config.LoadConfig().PyConverterPath,
		videoID,
	}
	if isAudio {
		args = append(args, "audio", config.LoadConfig().StoragePath)
	} else {
		args = append(args, "video", config.LoadConfig().StoragePath, "--resolution", resolution)
	}

	if cookiesPath != "" {
		args = append(args, "--cookies", cookiesPath)
		fmt.Println("Usando archivo de cookies en:", cookiesPath)
	}

	// Mostrar comando por consola
	fmt.Println("Ejecutando comando:", "/usr/bin/python3", strings.Join(args, " "))

	// Ejecutar comando
	cmd := exec.Command("/usr/bin/python3", args...)
	output, err := cmd.Output()
	if err != nil {
		// Fallo: marcar en base de datos
		_, updateErr := db.DB.Exec("UPDATE video_status SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE video_id = ? AND resolution = ?", "failed", videoID, resolution)
		if updateErr != nil {
			fmt.Printf("Error actualizando estado a failed: %v\n", updateErr)
		}
		return "", fmt.Errorf("error al ejecutar el comando: %v, output: %v", err, string(output))
	}

	// Obtener la última línea del output
	outputLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(outputLines) == 0 {
		return "", fmt.Errorf("no se obtuvo output del comando")
	}
	videoPath := outputLines[len(outputLines)-1]

	// Si contiene "Error" en el output, se marca como fallido
	if strings.Contains(videoPath, "Error") || strings.Contains(videoPath, "ERROR") {
		_, updateErr := db.DB.Exec("UPDATE video_status SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE video_id = ? AND resolution = ?", "failed", videoID, resolution)
		if updateErr != nil {
			fmt.Printf("Error actualizando estado a failed: %v\n", updateErr)
		}
		return "", fmt.Errorf("error procesando el video")
	}

	// Guardar estado exitoso con la ruta del archivo
	_, err = db.DB.Exec("UPDATE video_status SET status = ?, path = ?, updated_at = CURRENT_TIMESTAMP WHERE video_id = ? AND resolution = ?", "completed", videoPath, videoID, resolution)
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
