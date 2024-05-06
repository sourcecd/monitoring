package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sourcecd/monitoring/internal/metrictypes"
)

const (
	checkExistsQuery = "select exists ( select from information_schema.tables where table_name = 'monitoring' )"
	populateQuery    = "create table monitoring ( id varchar(64) PRIMARY KEY, mtype varchar(16), delta integer, value double precision )"
)

type PgDB struct {
	db      *sql.DB
	timeout time.Duration
}

func NewPgDB(dsn string) (*PgDB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PgDB{db: db, timeout: 60 * time.Second}, nil
}

func (p *PgDB) PopulateDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	var exists bool
	row := p.db.QueryRowContext(ctx, checkExistsQuery)
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("exists check failed: %s", err.Error())
	}
	if !exists {
		if _, err := p.db.ExecContext(ctx, populateQuery); err != nil {
			return fmt.Errorf("populate failed: %s", err.Error())
		}
	}
	return nil
}

func (p *PgDB) WriteGauge(name string, value metrictypes.Gauge) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
	values ($1, $2, $3) on conflict (id) do update set value = $3`, name, "gauge", value); err != nil {
		return fmt.Errorf("write gauge to db failed: %s", err.Error())
	}
	return nil
}
func (p *PgDB) WriteCounter(name string, value metrictypes.Counter) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
	values ($1, $2, $3) on conflict (id) 
	do update set delta = $3 + (select delta from monitoring where id = $1)`, name, "counter", value); err != nil {
		return fmt.Errorf("write counter to db failed: %s", err.Error())
	}
	return nil
}
func (p *PgDB) GetGauge(name string) (metrictypes.Gauge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	var gauge float64
	row := p.db.QueryRowContext(ctx, "select value from monitoring where id = $1", name)
	if err := row.Scan(&gauge); err != nil {
		return metrictypes.Gauge(0), fmt.Errorf("get gauge failed: %s", err.Error())
	}
	return metrictypes.Gauge(gauge), nil
}
func (p *PgDB) GetCounter(name string) (metrictypes.Counter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	var counter int64
	row := p.db.QueryRowContext(ctx, "select delta from monitoring where id = $1", name)
	if err := row.Scan(&counter); err != nil {
		return metrictypes.Counter(0), fmt.Errorf("get counter failed: %s", err.Error())
	}
	return metrictypes.Counter(counter), nil
}
func (p *PgDB) GetAllMetricsTxt() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	s := "---Counters---\n"
	var id string
	var delta int64
	var value float64
	rowsc, err := p.db.QueryContext(ctx, "select id, delta from monitoring where mtype = 'counter'")
	if err != nil {
		return "", err
	}
	defer rowsc.Close()
	for rowsc.Next() {
		if err := rowsc.Scan(&id, &delta); err != nil {
			return "", err
		}
		s += fmt.Sprintf("%v: %v\n", id, delta)
	}
	if rowsc.Err() != nil {
		return "", err
	}
	s += "---Gauge---\n"
	rowsg, err := p.db.QueryContext(ctx, "select id, value from monitoring where mtype = 'gauge'")
	if err != nil {
		return "", err
	}
	defer rowsg.Close()
	for rowsg.Next() {
		if err := rowsg.Scan(&id, &value); err != nil {
			return "", err
		}
		s += fmt.Sprintf("%v: %v\n", id, value)
	}
	if rowsg.Err() != nil {
		return "", err
	}

	return s, nil
}
func (p *PgDB) GetMetric(mType, name string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	var value float64
	var delta int64
	if mType == metrictypes.GaugeType {
		row := p.db.QueryRowContext(ctx, "select value from monitoring where id = $1", name)
		if err := row.Scan(&value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("no value")
			}
			return nil, err
		}
		return metrictypes.Gauge(value), nil
	} else if mType == metrictypes.CounterType {
		row := p.db.QueryRowContext(ctx, "select delta from monitoring where id = $1", name)
		if err := row.Scan(&delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("no value")
			}
			return nil, err
		}
		return metrictypes.Counter(delta), nil
	} else {
		return nil, errors.New("bad metric type")
	}
}

func (p *PgDB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	return p.db.PingContext(ctx)
}
func (p *PgDB) CloseDB() error {
	return p.db.Close()
}
func (p *PgDB) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}
