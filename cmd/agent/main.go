package main

import (
	"github.com/sourcecd/monitoring/internal/agent"
)

func main() {

	servFlags()
	servEnv()

	agent.Run(serverAddr, reportInterval, pollInterval)

}
