package main

import "flag"

var serverAddr string

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.Parse()
}
