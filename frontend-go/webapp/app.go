package webapp

import (
	"google.golang.org/grpc"
)

// App struct
type App struct {
	HTMLDir   string
	StaticDir string
	Conn      *grpc.ClientConn
}

// HTMLData struct for passing data to html templates
type HTMLData struct {
	Data  string
	Error string
	Stats Statistics
}

// Statistics defines the Code:URL structure
type Statistics struct {
	Code         string `json:"code"`
	URL          string `json:"url"`
	CreatedAt    string `json:"created_at"`
	LastAccessed string `json:"last_accessed"`
	Clicks       string `json:"clicks"`
}
