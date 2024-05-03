package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sourcecd/monitoring/internal/metrictypes"
)

type PgDb struct {
	db *sql.DB
}

func NewPgDb(host string) (*PgDb, error) {
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s", host))
	if err != nil {
		return nil, err
	}
	return &PgDb{db: db}, nil
}

func (p *PgDb) WriteGauge(name string, value metrictypes.Gauge) error {
	return nil
}
func (p *PgDb) WriteCounter(name string, value metrictypes.Counter) error {
	return nil
}
func (p *PgDb) GetGauge(name string) (metrictypes.Gauge, error) {
	return metrictypes.Gauge(0), nil
}
func (p *PgDb) GetCounter(name string) (metrictypes.Counter, error) {
	return metrictypes.Counter(0), nil
}
func (p *PgDb) GetAllMetricsTxt() string {
	return ""
}
func (p *PgDb) GetMetric(mType, name string) (interface{}, error) {
	return nil, nil
}

func (p *PgDb) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
