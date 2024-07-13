package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sourcecd/monitoring/internal/server"

	"net/http"
	//profile
	_ "net/http/pprof"
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

	//profile
	if config.PprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(config.PprofAddr, nil))
		}()
	}

	server.Run(ctx, config)
}
