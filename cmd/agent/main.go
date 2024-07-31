// Agent command for sending monitoring metrics.
// Uses http api.
package main

import (
	"log"

	"github.com/sourcecd/monitoring/internal/agent"

	"net/http"
	// Profile module.
	_ "net/http/pprof"
)

func main() {
	// Print Build args
	printBuildFlags()

	// Main config.
	var config agent.ConfigArgs

	// Parse cmdline flags.
	servFlags(&config)
	// Parse env options.
	servEnv(&config)

	// Enable profile server.
	if config.PprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(config.PprofAddr, nil))
		}()
	}

	// Run main program.
	agent.Run(config)

}
