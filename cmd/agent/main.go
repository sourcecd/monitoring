package main

import (
	"log"

	"github.com/sourcecd/monitoring/internal/agent"

	//profile
	"net/http"
	_ "net/http/pprof"
)

func main() {
	var config agent.ConfigArgs

	servFlags(&config)
	servEnv(&config)

	//profile
	if config.PprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(config.PprofAddr, nil))
		}()
	}

	agent.Run(config)

}
