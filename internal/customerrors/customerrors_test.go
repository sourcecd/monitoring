package customerrors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXxx(t *testing.T) {
	require.Equal(t, ErrNoVal.Error(), "no value")
	require.Equal(t, ErrBadMetricType.Error(), "bad metric type")
	require.Equal(t, ErrWrongMetricType.Error(), "wrong metric type")
	require.Equal(t, ErrWrongMetricValueType.Error(), "wrong metric value type")
}
