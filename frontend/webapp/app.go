package webapp

import (
	"google.golang.org/grpc"
)

// App holds the frontend application dependencies.
type App struct {
	SPADir     string           // React SPA build output directory
	BackendURL string           // Backend REST gateway URL for API proxying
	Conn       *grpc.ClientConn // gRPC connection to backend
}
