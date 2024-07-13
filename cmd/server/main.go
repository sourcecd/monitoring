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
	//profile
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	
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
