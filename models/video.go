package models

type Video struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id"`
	VideoID       string `json:"video_id"`
	Title         string `json:"title"`
	RequestedByIP string `json:"requested_by_ip"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}
