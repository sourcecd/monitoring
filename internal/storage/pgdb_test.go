package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
	"github.com/stretchr/testify/require"
)

var (
	db   *sql.DB
	mock sqlmock.Sqlmock
	err  error
	pgdb *PgDB
)

func TestCreatePGDB(t *testing.T) {
	ctx := context.Background()
	db, mock, err = sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	//t.Cleanup(func() {db.Close()})

	pgdb, err = NewPgDB("", db)
	require.NoError(t, err)

	mock.ExpectExec(populateQuery).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(getGaugePrep)
	mock.ExpectPrepare(getCounterPrep)
	mock.ExpectPrepare(getAllGaugePrep)
	mock.ExpectPrepare(getAllCounterPrep)
	mock.ExpectPrepare(insertGaugePrep)
	mock.ExpectPrepare(insertCounterPrep)

	err = pgdb.PopulateDB(ctx)
	require.NoError(t, err)
	err = pgdb.PopulateDB(ctx)
	require.Error(t, err)
}

func TestWriteMetricPG(t *testing.T) {
	ctx := context.Background()

	mock.ExpectExec(insertGaugePrep).WithArgs("testGauge", "gauge", 0.1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertCounterPrep).WithArgs("testCounter", "counter", 1).WillReturnResult(sqlmock.NewResult(1, 1))

	err = pgdb.WriteMetric(ctx, "gauge", "testGauge", metrictypes.Gauge(0.1))
	require.NoError(t, err)
	err = pgdb.WriteMetric(ctx, "counter", "testCounter", metrictypes.Counter(1))
	require.NoError(t, err)

	err = pgdb.WriteMetric(ctx, "gauge", "testGauge2", metrictypes.Gauge(0.1))
	require.Error(t, err)
	err = pgdb.WriteMetric(ctx, "counter", "testCounter2", metrictypes.Counter(1))
	require.Error(t, err)
	err = pgdb.WriteMetric(ctx, "gauge", "testGauge2", 0.1)
	require.Error(t, err)
	err = pgdb.WriteMetric(ctx, "counter", "testCounter2", 1)
	require.Error(t, err)

	err = pgdb.WriteMetric(ctx, "wrong", "testCounter2", 1)
	require.Error(t, err)
}

func TestWriteBatchMetricsPG(t *testing.T) {
	ctx := context.Background()
	d := int64(1)
	v := float64(0.1)
	m := []models.Metrics{
		{
			Delta: &d,
			ID:    "testCounter1",
			MType: "counter",
		},
		{
			Value: &v,
			ID:    "testGauge1",
			MType: "gauge",
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(insertCounterPrep).WithArgs("testCounter1", "counter", 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertGaugePrep).WithArgs("testGauge1", "gauge", 0.1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = pgdb.WriteBatchMetrics(ctx, m)
	require.NoError(t, err)
}

func TestGetAllMetricsTxt(t *testing.T) {
	ctx := context.Background()

	expRes := `---Counters---
testCounter2: 1
---Gauge---
testGauge2: 0.1
`

	mock.ExpectQuery(getAllCounterPrep).WillReturnRows(sqlmock.NewRows([]string{"id", "delta"}).AddRow("testCounter2", 1))
	mock.ExpectQuery(getAllGaugePrep).WillReturnRows(sqlmock.NewRows([]string{"id", "value"}).AddRow("testGauge2", 0.1))

	st, err := pgdb.GetAllMetricsTxt(ctx)
	require.NoError(t, err)
	require.Equal(t, expRes, st)
}

func TestGetMetric(t *testing.T) {
	ctx := context.Background()

	mock.ExpectQuery(getGaugePrep).WithArgs("testGauge3").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(0.1))
	mock.ExpectQuery(getCounterPrep).WithArgs("testCounter3").WillReturnRows(sqlmock.NewRows([]string{"delta"}).AddRow(1))

	i, err := pgdb.GetMetric(ctx, "gauge", "testGauge3")
	require.NoError(t, err)
	require.Equal(t, metrictypes.Gauge(0.1), i.(metrictypes.Gauge))
	i, err = pgdb.GetMetric(ctx, "counter", "testCounter3")
	require.NoError(t, err)
	require.Equal(t, metrictypes.Counter(1), i.(metrictypes.Counter))

	_, err = pgdb.GetMetric(ctx, "counter", "testCounter3")
	require.Error(t, err)
}

func TestPingPG(t *testing.T) {
	ctx := context.Background()

	mock.ExpectPing()

	err = pgdb.Ping(ctx)
	require.NoError(t, err)
}

func TestClosePG(t *testing.T) {

	mock.ExpectClose()

	err = pgdb.CloseDB()
	require.NoError(t, err)
}
