package main

import "flag"

var (
	serverAddr string
	reportInterval int
	pollInterval int
)

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&reportInterval, "r", 10, "metrics report interval")
	flag.IntVar(&pollInterval, "p", 2, "metrics poll interval")
	flag.Parse()
}