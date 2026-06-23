package main

import (
	"flag"
	"log"
	"os"
)

type StdioServer struct {
	// Placeholder for stdio mode server
}

func NewServer() *StdioServer {
	return &StdioServer{}
}

func (s *StdioServer) Start() {
	log.Println("[Stdio Server] Stdio mode not implemented yet")
}

func main() {
	var mode string
	var addr string

	flag.StringVar(&mode, "mode", "http", "Server mode: http or stdio")
	flag.StringVar(&addr, "addr", ":8080", "HTTP server address (for http mode)")
	flag.Parse()

	if mode == "http" {
		// HTTP Server Mode (for cloud deployment)
		server := NewHTTPServer(addr)
		if err := server.Start(); err != nil {
			log.Fatalf("[ERROR] HTTP Server failed: %v", err)
		}
	} else if mode == "stdio" {
		// Stdio Mode (for local IPC)
		server := NewServer()
		server.Start()
	} else {
		log.Fatalf("Unknown mode: %s (use 'http' or 'stdio')", mode)
		os.Exit(1)
	}
}
