package main

import (
	"flag"
	"os"
	"strings"
)

var serverAddr string
var loglevel string

func servEnv() {
	s := os.Getenv("ADDRESS")
	l := os.Getenv("LOG_LEVEL")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			serverAddr = s
		}
	}
	if l != "" {
		loglevel = l
	}
}

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&loglevel, "l", "info", "Log level for server")
	flag.Parse()
}
