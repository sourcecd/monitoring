package storage

import (
	"context"
	"os"
	"testing"

	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/stretchr/testify/require"
)

func TestFileWrite(t *testing.T) {
	tmpFile := "test_save_to_file.tmp"
	defer os.Remove(tmpFile)

	ctx := context.Background()
	memStorage := NewMemStorage()

	memStorage.WriteMetric(ctx, "gauge", "testmetric1", metrictypes.Gauge(0.1))
	memStorage.WriteMetric(ctx, "counter", "testmetric2", metrictypes.Counter(1))

	err := memStorage.SaveToFile(tmpFile)
	require.NoError(t, err)

	err = memStorage.ReadFromFile(tmpFile)
	require.NoError(t, err)
}
