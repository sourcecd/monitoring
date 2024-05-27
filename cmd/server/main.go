package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sourcecd/monitoring/internal/server"
)

// seconds
const interruptAfter = 10

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	context.AfterFunc(ctx, func() {
		time.Sleep(interruptAfter * time.Second)
		log.Fatal("Interrupted by shutdown time exeeded!!!")
	})

	var config server.ConfigArgs

	servFlags(&config)
	servEnv(&config)

	server.Run(ctx, config)
}
