package main

import (
	"github.com/sourcecd/monitoring/internal/server"
)

func main() {

	servFlags()
	servEnv()

	server.Run(serverAddr)
}
