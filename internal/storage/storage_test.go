package storage

import (
	"context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
	"github.com/sourcecd/monitoring/mocks"
	"github.com/stretchr/testify/require"
)

func TestFileWrite(t *testing.T) {
	tmpFile := "test_save_to_file.tmp"
	t.Cleanup(func() { os.Remove(tmpFile) })

	ctx := context.Background()
	memStorage := NewMemStorage()

	memStorage.WriteMetric(ctx, "gauge", "testmetric1", metrictypes.Gauge(0.1))
	memStorage.WriteMetric(ctx, "counter", "testmetric2", metrictypes.Counter(1))

	err := memStorage.SaveToFile(tmpFile)
	require.NoError(t, err)

	err = memStorage.ReadFromFile(tmpFile)
	require.NoError(t, err)
}

func TestPing(t *testing.T) {
	ctx := context.Background()
	memStorage := NewMemStorage()
	err := memStorage.Ping(ctx)
	require.NoError(t, err)
}

func TestWriteBatchMetrics(t *testing.T) {
	ctx := context.Background()
	memStorage := NewMemStorage()
	d := int64(1)
	f := float64(0.1)
	metrics := []models.Metrics{
		{
			Delta: &d,
			ID:    "testCounter",
			MType: "counter",
		},
		{
			Value: &f,
			ID:    "testGauge",
			MType: "gauge",
		},
	}

	err := memStorage.WriteBatchMetrics(ctx, metrics)
	require.NoError(t, err)
}

func TestAllGoMocks(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mDB := mocks.NewMockStoreMetrics(ctrl)

	mDB.EXPECT().GetAllMetricsTxt(gomock.Any()).Return("test", nil)
	mDB.EXPECT().GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(gomock.Any(), nil)
	mDB.EXPECT().WriteBatchMetrics(gomock.Any(), gomock.Any()).Return(nil)
	mDB.EXPECT().WriteMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	mDB.GetAllMetricsTxt(ctx)
	mDB.GetMetric(ctx, "test1", "test2")
	mDB.WriteBatchMetrics(ctx, []models.Metrics{})
	mDB.WriteMetric(ctx, "test3", "test4", "ok")
}
