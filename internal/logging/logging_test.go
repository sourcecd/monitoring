package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogging(t *testing.T) {
	err := Setup("info")
	require.NoError(t, err)
}
