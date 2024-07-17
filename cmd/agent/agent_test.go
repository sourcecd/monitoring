package main

import (
	"os"
	"testing"

	"github.com/sourcecd/monitoring/internal/agent"
	"github.com/stretchr/testify/require"
)

func TestAgentCmdArgs(t *testing.T) {
	var config agent.ConfigArgs
	// set some args for cmdline check
	os.Args = append(os.Args, "-a", "localhost:8080")
	os.Args = append(os.Args, "-r", "10")
	os.Args = append(os.Args, "-p", "2")
	os.Args = append(os.Args, "-l", "1")
	os.Args = append(os.Args, "-k", "seckey")
	os.Args = append(os.Args, "-d", "localhost:6060")

	servFlags(&config)

	// check flags
	require.Equal(t, config.ServerAddr, "localhost:8080")
	require.Equal(t, config.ReportInterval, 10)
	require.Equal(t, config.PollInterval, 2)
	require.Equal(t, config.RateLimit, 1)
	require.Equal(t, config.KeyEnc, "seckey")
	require.Equal(t, config.PprofAddr, "localhost:6060")
}

func TestAgentEnvArgs(t *testing.T) {
	var config agent.ConfigArgs
	// set test env args
	os.Setenv("ADDRESS", "localhost:9090")
	os.Setenv("REPORT_INTERVAL", "20")
	os.Setenv("POLL_INTERVAL", "6")
	os.Setenv("RATE_LIMIT", "3")
	os.Setenv("PPROF_AGENT_ADDRESS", "localhost:7070")
	os.Setenv("KEY", "seckey2")

	servEnv(&config)

	// check env args
	require.Equal(t, config.ServerAddr, "localhost:9090")
	require.Equal(t, config.ReportInterval, 20)
	require.Equal(t, config.PollInterval, 6)
	require.Equal(t, config.RateLimit, 3)
	require.Equal(t, config.KeyEnc, "seckey2")
	require.Equal(t, config.PprofAddr, "localhost:7070")
}
