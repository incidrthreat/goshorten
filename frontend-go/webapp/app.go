package webapp

import "google.golang.org/grpc"

// App ...
type App struct {
	HTMLDir   string
	StaticDir string
	Conn      *grpc.ClientConn
}
