package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sourcecd/monitoring/internal/agent"
)

func servEnv(config *agent.ConfigArgs) {
	s := os.Getenv("ADDRESS")
	r := os.Getenv("REPORT_INTERVAL")
	p := os.Getenv("POLL_INTERVAL")
	k := os.Getenv("KEY")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			config.ServerAddr = s
		}
	}
	if r != "" {
		i, err := strconv.Atoi(r)
		if err != nil {
			log.Fatal(err)
		}
		config.ReportInterval = i
	}
	if p != "" {
		i, err := strconv.Atoi(p)
		if err != nil {
			log.Fatal(err)
		}
		config.PollInterval = i
	}
	if k != "" {
		config.KeyEnc = k
	}
}

func servFlags(config *agent.ConfigArgs) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&config.ReportInterval, "r", 10, "metrics report interval")
	flag.IntVar(&config.PollInterval, "p", 2, "metrics poll interval")
	flag.StringVar(&config.KeyEnc, "k", "", "encrypted key")
	flag.Parse()
}
