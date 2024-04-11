package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

var (
	serverAddr     string
	reportInterval int
	pollInterval   int
)

func servEnv() {
	s := os.Getenv("ADDRESS")
	r := os.Getenv("REPORT_INTERVAL")
	p := os.Getenv("POLL_INTERVAL")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			serverAddr = s
		}
	}
	if r != "" {
		i, err := strconv.Atoi(r)
		if err == nil {
			reportInterval = i
		}
	}
	if p != "" {
		i, err := strconv.Atoi(p)
		if err == nil {
			pollInterval = i
		}
	}
}

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&reportInterval, "r", 10, "metrics report interval")
	flag.IntVar(&pollInterval, "p", 2, "metrics poll interval")
	flag.Parse()
}
