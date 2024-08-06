package main

import (
	"io"
	"os"
	"testing"

	"github.com/sourcecd/monitoring/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCmdArgs(t *testing.T) {
	t.Parallel()
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
	assert.Equal(t, config.ServerAddr, "localhost:8080")
	assert.Equal(t, config.ReportInterval, 10)
	assert.Equal(t, config.PollInterval, 2)
	assert.Equal(t, config.RateLimit, 1)
	assert.Equal(t, config.KeyEnc, "seckey")
	assert.Equal(t, config.PprofAddr, "localhost:6060")
}

func TestAgentEnvArgs(t *testing.T) {
	t.Parallel()
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
	assert.Equal(t, config.ServerAddr, "localhost:9090")
	assert.Equal(t, config.ReportInterval, 20)
	assert.Equal(t, config.PollInterval, 6)
	assert.Equal(t, config.RateLimit, 3)
	assert.Equal(t, config.KeyEnc, "seckey2")
	assert.Equal(t, config.PprofAddr, "localhost:7070")
}

func TestBuildOpts(t *testing.T) {
	t.Parallel()
	var err error
	testF := "test_build_opts_agent.tmp"

	expString := `Build version: 1
Build date: 1970year
Build commit: testOk
`

	stdo := os.Stdout
	os.Stdout, err = os.OpenFile(testF, os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(testF) })

	buildVersion = "1"
	buildDate = "1970year"
	buildCommit = "testOk"

	printBuildFlags()

	os.Stdout.Close()
	os.Stdout = stdo

	f, err := os.Open(testF)
	require.NoError(t, err)
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, expString, string(b))
}
