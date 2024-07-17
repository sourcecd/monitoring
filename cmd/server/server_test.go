package main

import (
	"os"
	"testing"

	"github.com/sourcecd/monitoring/internal/server"
	"github.com/stretchr/testify/require"
)

func TestServerCmdArgs(t *testing.T) {
	var config server.ConfigArgs
	// set some args for cmdline check
	os.Args = append(os.Args, "-a", "localhost:8080")
	os.Args = append(os.Args, "-l", "info")
	os.Args = append(os.Args, "-i", "300")
	os.Args = append(os.Args, "-f", "/tmp/metrics-db.json")
	os.Args = append(os.Args, "-r")
	os.Args = append(os.Args, "-d", "host=localhost database=monitoring")
	os.Args = append(os.Args, "-k", "seckey")
	os.Args = append(os.Args, "-p", "localhost:6060")

	servFlags(&config)

	// check flags
	require.Equal(t, config.ServerAddr, "localhost:8080")
	require.Equal(t, config.Loglevel, "info")
	require.Equal(t, config.StoreInterval, 300)
	require.Equal(t, config.FileStoragePath, "/tmp/metrics-db.json")
	require.Equal(t, config.Restore, true)
	require.Equal(t, config.DatabaseDsn, "host=localhost database=monitoring")
	require.Equal(t, config.KeyEnc, "seckey")
	require.Equal(t, config.PprofAddr, "localhost:6060")
}

func TestServerEnvArgs(t *testing.T) {
	var config server.ConfigArgs
	// set test env args
	os.Setenv("ADDRESS", "localhost:9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("STORE_INTERVAL", "600")
	os.Setenv("FILE_STORAGE_PATH", "/home/metric.json")
	os.Setenv("RESTORE", "false")
	os.Setenv("DATABASE_DSN", "database=monitoring")
	os.Setenv("PPROF_SERVER_ADDRESS", "localhost:7070")
	os.Setenv("KEY", "seckey2")

	servEnv(&config)

	// check env args
	require.Equal(t, config.ServerAddr, "localhost:9090")
	require.Equal(t, config.Loglevel, "debug")
	require.Equal(t, config.StoreInterval, 600)
	require.Equal(t, config.FileStoragePath, "/home/metric.json")
	require.Equal(t, config.Restore, false)
	require.Equal(t, config.DatabaseDsn, "database=monitoring")
	require.Equal(t, config.KeyEnc, "seckey2")
	require.Equal(t, config.PprofAddr, "localhost:7070")
}
