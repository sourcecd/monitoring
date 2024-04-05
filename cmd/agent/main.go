package main

import (
	"github.com/sourcecd/monitoring/internal/agent"
)

func main() {

	servFlags()

	agent.Run(serverAddr, reportInterval, pollInterval)

}
