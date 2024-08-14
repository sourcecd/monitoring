// Agent command for sending monitoring metrics.
// Uses http api.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourcecd/monitoring/internal/agent"

	"net/http"
	// Profile module.
	_ "net/http/pprof"
)

// Number of seconds before force interrupt program.
const interruptAfter = 30

func main() {
	// Print Build args
	printBuildFlags()

	// Context for using gracefull shutdown with interrupt signal.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	// Context run func when context.Done resived.
	context.AfterFunc(ctx, func() {
		log.Println("received gracefull shutdown signal")
		time.Sleep(interruptAfter * time.Second)
		log.Fatal("Interrupted by shutdown time exeeded!!!")
	})

	// Main config.
	var config agent.ConfigArgs

	// Parse cmdline flags.
	servFlags(&config)
	// Parse env options.
	servEnv(&config)
	// Parse json config
	parseJSONconfigFile(&config)

	// Enable profile server.
	if config.PprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(config.PprofAddr, nil))
		}()
	}

	// Run main program.
	agent.Run(ctx, config)

}
