package main

import (
	"fmt"
	"github.com/Daniel-W-Innes/street_view_proxy/config"
	"github.com/Daniel-W-Innes/street_view_proxy/servers"
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", config.Port))
	if err != nil {
		log.Fatalf("failed to listen on %d: %v", config.Port, err)
	}
	var opts []grpc.ServerOption

	server := grpc.NewServer(opts...)
	view.RegisterImageDownloaderServer(server, &servers.ImageDownloaderServer{ApiKey: os.Getenv("API_KEY")})
	fmt.Printf("listen on port %d\n", config.Port)
	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve server: %v", err)
	}
}
