package storage

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sourcecd/monitoring/internal/metrictypes"
)

type PgDB struct {
	db *sql.DB
}

func NewPgDB(dsn string) (*PgDB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PgDB{db: db}, nil
}

func (p *PgDB) WriteGauge(name string, value metrictypes.Gauge) error {
	return nil
}
func (p *PgDB) WriteCounter(name string, value metrictypes.Counter) error {
	return nil
}
func (p *PgDB) GetGauge(name string) (metrictypes.Gauge, error) {
	return metrictypes.Gauge(0), nil
}
func (p *PgDB) GetCounter(name string) (metrictypes.Counter, error) {
	return metrictypes.Counter(0), nil
}
func (p *PgDB) GetAllMetricsTxt() string {
	return ""
}
func (p *PgDB) GetMetric(mType, name string) (interface{}, error) {
	return nil, nil
}

func (p *PgDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
