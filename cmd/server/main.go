package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sourcecd/monitoring/internal/server"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var config server.ConfigArgs

	servFlags(&config)
	servEnv(&config)

	server.Run(config, sigs)
}
