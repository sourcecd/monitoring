package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

const populateQuery = `create table if not exists monitoring ( id varchar(64) PRIMARY KEY, 
mtype varchar(16), delta bigint, value double precision )`

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
	if _, err := p.db.ExecContext(ctx, populateQuery); err != nil {
		return fmt.Errorf("populate failed: %s", err.Error())
	}
	return nil
}

func (p *PgDB) WriteMetric(mtype, name string, val interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	if mtype == metrictypes.GaugeType {
		if metric, ok := val.(metrictypes.Gauge); ok {
			if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
			values ($1, $2, $3) on conflict (id) do update set value = $3`, name, "gauge", metric); err != nil {
				return fmt.Errorf("write gauge to db failed: %s", err.Error())
			}
			return nil
		}
		return errors.New("wrong metric value type")
	} else if mtype == metrictypes.CounterType {
		if metric, ok := val.(metrictypes.Counter); ok {
			if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
			values ($1, $2, $3) on conflict (id) 
			do update set delta = $3 + (select delta from monitoring where id = $1)`, name, "counter", metric); err != nil {
				return fmt.Errorf("write counter to db failed: %s", err.Error())
			}
			return nil
		}
		return errors.New("wrong metric value type")
	}
	return errors.New("wrong metric type")
}
func (p *PgDB) WriteBatchMetrics(metrics []models.Metrics) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("can't start tx to db: %s", err.Error())
	}
	defer tx.Rollback()
	for _, v := range metrics {
		if v.MType == metrictypes.GaugeType && v.Value != nil {
			if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
			values ($1, $2, $3) on conflict (id) do update set value = $3`, v.ID, v.MType, v.Value); err != nil {
				return fmt.Errorf("write gauge to db failed: %s", err.Error())
			}
		} else if v.MType == metrictypes.CounterType && v.Delta != nil {
			if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
			values ($1, $2, $3) on conflict (id) 
			do update set delta = $3 + (select delta from monitoring where id = $1)`, v.ID, v.MType, v.Delta); err != nil {
				return fmt.Errorf("write counter to db failed: %s", err.Error())
			}
		} else {
			return errors.New("wrong metric type or nil value")
		}
	}
	return tx.Commit()
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
