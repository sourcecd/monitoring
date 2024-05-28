package main

import (
	"github.com/sourcecd/monitoring/internal/agent"
)

func main() {
	var config agent.ConfigArgs

	servFlags(&config)
	servEnv(&config)

	agent.Run(config)

}
