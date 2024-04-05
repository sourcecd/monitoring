package main

import (
	"github.com/sourcecd/monitoring/internal/server"
)

func main() {

	servFlags()

	server.Run(serverAddr)
}
