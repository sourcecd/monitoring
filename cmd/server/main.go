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

	servFlags()
	servEnv()

	server.Run(serverAddr, loglevel, storeInterval, fileStoragePath, restore, sigs, databaseDsn)
}
