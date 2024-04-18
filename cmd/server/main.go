package main

import (
	"github.com/sourcecd/monitoring/internal/server"
)

const loglevel = "info"

func main() {

	servFlags()
	servEnv()

	server.Run(serverAddr, loglevel)
}
