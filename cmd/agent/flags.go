package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sourcecd/monitoring/internal/agent"
)

// Parse env args.
func servEnv(config *agent.ConfigArgs) {
	s := os.Getenv("ADDRESS")
	r := os.Getenv("REPORT_INTERVAL")
	p := os.Getenv("POLL_INTERVAL")
	k := os.Getenv("KEY")
	l := os.Getenv("RATE_LIMIT")
	d := os.Getenv("PPROF_AGENT_ADDRESS")
	c := os.Getenv("CRYPTO_KEY")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			config.ServerAddr = s
		}
	}
	if d != "" {
		if len(strings.Split(d, ":")) == 2 {
			config.PprofAddr = d
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
	if l != "" {
		i, err := strconv.Atoi(l)
		if err != nil {
			log.Fatal(err)
		}
		config.RateLimit = i
	}
	if c != "" {
		config.PubKeyFile = c
	}
}

// Parse cmdline args.
func servFlags(config *agent.ConfigArgs) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&config.ReportInterval, "r", 10, "metrics report interval")
	flag.IntVar(&config.PollInterval, "p", 2, "metrics poll interval")
	flag.StringVar(&config.KeyEnc, "k", "", "encrypted key")
	flag.IntVar(&config.RateLimit, "l", 1, "send ratelimit")
	flag.StringVar(&config.PprofAddr, "d", "", "pprof server bind addres and port")
	flag.StringVar(&config.PubKeyFile, "crypto-key", "", "path to public asymmetric key")
	flag.Parse()
}
