package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

var serverAddr string
var loglevel string
var storeInterval int
var fileStoragePath string
var restore bool
var databaseDsn string

func servEnv() {
	s := os.Getenv("ADDRESS")
	l := os.Getenv("LOG_LEVEL")
	i := os.Getenv("STORE_INTERVAL")
	f := os.Getenv("FILE_STORAGE_PATH")
	r := os.Getenv("RESTORE")
	d := os.Getenv("DATABASE_DSN")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			serverAddr = s
		}
	}
	if l != "" {
		loglevel = l
	}
	if i != "" {
		ii, err := strconv.Atoi(i)
		if err != nil {
			log.Fatal(err)
		}
		storeInterval = ii
	}
	if f != "" {
		fileStoragePath = f
	}
	if r != "" {
		b, err := strconv.ParseBool(r)
		if err != nil {
			log.Fatal(err)
		}
		restore = b
	}
	if d != "" {
		databaseDsn = d
	}
}

func servFlags() {
	flag.StringVar(&serverAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&loglevel, "l", "info", "Log level for server")
	flag.IntVar(&storeInterval, "i", 300, "metric store interval")
	flag.StringVar(&fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restore, "r", true, "restore metric data")
	flag.StringVar(&databaseDsn, "d", "localhost", "pg db connect address")
	flag.Parse()
}
