package models

type VideoStatus struct {
	ID         int    `json:"id"`
	VideoID    string `json:"video_id"`
	Resolution string `json:"resolution"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

const (
	Processing = "processing"
	Completed  = "completed"
	Failed     = "failed"
)
