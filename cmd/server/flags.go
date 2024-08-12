package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sourcecd/monitoring/internal/server"
)

// config file in json format
var cfgJSON string

// Parse json config file
func parseJSONconfigFile(config *server.ConfigArgs) {
	if cfgJSON == "" {
		return
	}
	f, err := os.Open(cfgJSON)
	if err != nil {
		log.Fatal(err)
	}
	jf, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(jf, &config); err != nil {
		log.Fatal(err)
	}
}

// Parse env args.
func servEnv(config *server.ConfigArgs) {
	s := os.Getenv("ADDRESS")
	l := os.Getenv("LOG_LEVEL")
	i := os.Getenv("STORE_INTERVAL")
	f := os.Getenv("FILE_STORAGE_PATH")
	r := os.Getenv("RESTORE")
	d := os.Getenv("DATABASE_DSN")
	p := os.Getenv("PPROF_SERVER_ADDRESS")
	k := os.Getenv("KEY")
	c := os.Getenv("CRYPTO_KEY")
	cfg := os.Getenv("CONFIG")

	if s != "" {
		if len(strings.Split(s, ":")) == 2 {
			config.ServerAddr = s
		}
	}
	if p != "" {
		if len(strings.Split(p, ":")) == 2 {
			config.PprofAddr = p
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
	if c != "" {
		config.PrivKeyFile = c
	}
	if cfg != "" {
		cfgJSON = cfg
	}
}

// Parse cmdline args.
func servFlags(config *server.ConfigArgs) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&config.Loglevel, "l", "info", "Log level for server")
	flag.IntVar(&config.StoreInterval, "i", 300, "metric store interval")
	flag.StringVar(&config.FileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&config.Restore, "r", true, "restore metric data")
	//dsn example: host=localhost database=monitoring
	flag.StringVar(&config.DatabaseDsn, "d", "", "pg db connect address")
	flag.StringVar(&config.KeyEnc, "k", "", "encrypted key")
	flag.StringVar(&config.PprofAddr, "p", "", "Pprof server bind addres and port")
	flag.StringVar(&config.PrivKeyFile, "crypto-key", "", "path to private asymmetric key")
	flag.StringVar(&cfgJSON, "config", "", "path to main config file (json)")
	flag.Parse()
}
