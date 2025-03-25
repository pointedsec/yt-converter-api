package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"yt-converter-api/config"
)

type YoutubeResponse struct {
	Items []struct {
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

// Comprueba que la URL es válida
func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// Función para comprobar si la URL de YouTube es válida y existe
func IsYoutubeUrl(url string) (bool, error) {
	// Validar que la URL es un enlace de YouTube
	youtubeRegex := `^(https?://)?(www\.)?(youtube|youtu|youtube-nocookie)\.(com|be)/(watch\?v=|embed\/|v\/|e\/|.*[?&]v=)[A-Za-z0-9_-]{11}$`
	re := regexp.MustCompile(youtubeRegex)

	// Si la URL no coincide con el patrón de YouTube, devolver false
	if !re.MatchString(url) {
		return false, fmt.Errorf("URL no es un enlace de YouTube válido")
	}

	// Hacer una solicitud HTTP a la URL del video
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("Error al hacer la solicitud HTTP: %v", err)
	}
	defer resp.Body.Close()

	// Comprobar el código de estado HTTP
	if resp.StatusCode == 200 {
		// Si el código de estado es 200, el video existe
		return true, nil
	}

	// Si el código de estado no es 200, el video no existe o hay algún error
	return false, fmt.Errorf("El video no existe o no está disponible, código de estado: %d", resp.StatusCode)
}

// Obtener el ID del video
func GetYoutubeVideoID(url string) string {
	re := regexp.MustCompile(`(?:v=|\/)([0-9A-Za-z_-]{11}).*`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Obtener el título del video en base a la URL (Usando Google Cloud API)
func GetYoutubeVideoTitle(videoURL string) (string, error) {
	videoID := GetYoutubeVideoID(videoURL)
	apiURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s", videoID, config.LoadConfig().GoogleCloudApiKey)

	// Hacer la solicitud HTTP a la API de YouTube
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("Error al hacer la solicitud HTTP a la API de YouTube: %v", err)
	}
	defer resp.Body.Close()

	// Leer la respuesta de la API
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error al leer la respuesta: %v", err)
	}

	// Deserializar la respuesta JSON
	var youtubeResp YoutubeResponse
	if err := json.Unmarshal(body, &youtubeResp); err != nil {
		return "", fmt.Errorf("Error al deserializar la respuesta: %v", err)
	}

	// Si encontramos un video en la respuesta, devolver el título
	if len(youtubeResp.Items) > 0 {
		return youtubeResp.Items[0].Snippet.Title, nil
	}

	// Si no se encuentra el video, devolver un error
	return "", fmt.Errorf("No se pudo encontrar el video con ID %s", videoID)
}

// Obtener las resoluciones de un video haciendo uso de pyConverter/main.py
func GetYoutubeVideoResolutions(videoID string) ([]string, error) {
	// Construir el comando para ejecutar el script de Python
	cmd := exec.Command("/usr/bin/python3", config.LoadConfig().PyConverterPath, videoID, "video", config.LoadConfig().StoragePath)

	// Capturar la salida del comando
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error al ejecutar el script de Python, comprueba la dirección URL del video u otros factores: %v, output: %v", err, string(output))
	}

	// Dividir la salida en líneas y obtener la última línea
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no se obtuvo salida del script de Python")
	}
	lastLine := lines[len(lines)-1]

	// Limpiar la cadena de caracteres innecesarios
	lastLine = strings.TrimPrefix(lastLine, "[")
	lastLine = strings.TrimSuffix(lastLine, "]")
	lastLine = strings.ReplaceAll(lastLine, "'", "")
	lastLine = strings.ReplaceAll(lastLine, " ", "")

	// Convertir la cadena en una lista de formatos
	formatList := strings.Split(lastLine, ",")
	if len(formatList) == 0 {
		return nil, fmt.Errorf("no se encontraron formatos disponibles")
	}

	return formatList, nil
}
