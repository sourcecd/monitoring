package main

import (
	"flag"
	"os"
	"strings"
)

var serverAddr string

func servEnv() {
	s := os.Getenv("ADDRESS")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			serverAddr = s
		}
	}
}

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.Parse()
}
