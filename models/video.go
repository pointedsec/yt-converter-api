package models

type Video struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id"`
	VideoID       string `json:"video_id"`
	Title         string `json:"title"`
	Format        string `json:"format"` // 'mp3' o 'mp4'
	Path          string `json:"path"`
	RequestedByIP string `json:"requested_by_ip"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}
