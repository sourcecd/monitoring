// Server command for collecting monitoring metrics.
// Uses http api.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sourcecd/monitoring/internal/server"

	"net/http"
	// Profile module.
	_ "net/http/pprof"
)

// Number of seconds before force interrupt program.
const interruptAfter = 10

func main() {
	// Print Build args
	printBuildFlags()

	// Context for using gracefull shutdown with interrupt signal.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Context run func when context.Done resived.
	context.AfterFunc(ctx, func() {
		time.Sleep(interruptAfter * time.Second)
		log.Fatal("Interrupted by shutdown time exeeded!!!")
	})

	// Main config.
	var config server.ConfigArgs

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
	server.Run(ctx, config)
}
