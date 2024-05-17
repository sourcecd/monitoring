package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sourcecd/monitoring/internal/server"
)

func servEnv(config *server.ConfigArgs) {
	s := os.Getenv("ADDRESS")
	l := os.Getenv("LOG_LEVEL")
	i := os.Getenv("STORE_INTERVAL")
	f := os.Getenv("FILE_STORAGE_PATH")
	r := os.Getenv("RESTORE")
	d := os.Getenv("DATABASE_DSN")
	k := os.Getenv("KEY")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			config.ServerAddr = s
		}
	}
	if l != "" {
		config.Loglevel = l
	}
	if i != "" {
		ii, err := strconv.Atoi(i)
		if err != nil {
			log.Fatal(err)
		}
		config.StoreInterval = ii
	}
	if f != "" {
		config.FileStoragePath = f
	}
	if r != "" {
		b, err := strconv.ParseBool(r)
		if err != nil {
			log.Fatal(err)
		}
		config.Restore = b
	}
	if d != "" {
		config.DatabaseDsn = d
	}
	if k != "" {
		config.KeyEnc = k
	}
}

func servFlags(config *server.ConfigArgs) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&config.Loglevel, "l", "info", "Log level for server")
	flag.IntVar(&config.StoreInterval, "i", 300, "metric store interval")
	flag.StringVar(&config.FileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&config.Restore, "r", true, "restore metric data")
	//dsn example: host=localhost database=monitoring
	flag.StringVar(&config.DatabaseDsn, "d", "", "pg db connect address")
	flag.StringVar(&config.KeyEnc, "k", "", "encrypted key")
	flag.Parse()
}
