package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sethvargo/go-retry"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

const populateQuery = `create table if not exists monitoring ( id varchar(64) PRIMARY KEY, 
mtype varchar(16), delta bigint, value double precision )`

type backOff struct {
	fiboDuration time.Duration
	maxRetries   uint64
}

type PgDB struct {
	db      *sql.DB
	timeout time.Duration
	backoff backOff
}

func NewPgDB(dsn string) (*PgDB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PgDB{db: db, timeout: 60 * time.Second, backoff: backOff{fiboDuration: 1 * time.Second, maxRetries: 3}}, nil
}

func (p *PgDB) PopulateDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	bf := retry.WithMaxRetries(p.backoff.maxRetries, retry.NewFibonacci(p.backoff.fiboDuration))

	if err := retry.Do(ctx, bf, func(ctx context.Context) error {
		if _, err := p.db.ExecContext(ctx, populateQuery); err != nil {
			return retry.RetryableError(fmt.Errorf("populate failed: %s", err.Error()))
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (p *PgDB) WriteMetric(mtype, name string, val interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	bf := retry.WithMaxRetries(p.backoff.maxRetries, retry.NewFibonacci(p.backoff.fiboDuration))

	switch mtype{
	case metrictypes.GaugeType:
		if metric, ok := val.(metrictypes.Gauge); ok {
			if err := retry.Do(ctx, bf, func(ctx context.Context) error {
				//idempotency
				if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
				values ($1, $2, $3) on conflict (id) do update set value = $3`, name, "gauge", metric); err != nil {
					return retry.RetryableError(fmt.Errorf("write gauge to db failed: %s", err.Error()))
				}
				return nil
			}); err != nil {
				return err
			}
			return nil
		}
		return errors.New("wrong metric value type")
	case metrictypes.CounterType:
		if metric, ok := val.(metrictypes.Counter); ok {
			if err := retry.Do(ctx, bf, func(ctx context.Context) error {
				if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
				values ($1, $2, $3) on conflict (id) 
				do update set delta = $3 + (select delta from monitoring where id = $1)`, name, "counter", metric); err != nil {
					return retry.RetryableError(fmt.Errorf("write counter to db failed: %s", err.Error()))
				}
				return nil
			}); err != nil {
				return err
			}
			return nil
		}
		return errors.New("wrong metric value type")
	default:
		return errors.New("wrong metric type")
	}
}
func (p *PgDB) WriteBatchMetrics(metrics []models.Metrics) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	var tx *sql.Tx
	var err error
	bf := retry.WithMaxRetries(p.backoff.maxRetries, retry.NewFibonacci(p.backoff.fiboDuration))

	if err = retry.Do(ctx, bf, func(ctx context.Context) error {
		tx, err = p.db.Begin()
		if err != nil {
			return retry.RetryableError(fmt.Errorf("can't start tx to db: %s", err.Error()))
		}
		return nil
	}); err != nil {
		return err
	}

	defer tx.Rollback()
	// i think we don't break all batch if one metric failed in batch (use continue)
	for _, v := range metrics {
		switch v.MType {
		case metrictypes.GaugeType:
			if v.Value == nil || v.ID == "" {
				log.Println("empty id or nil value gauge metric")
				continue
			}
			if err := retry.Do(ctx, bf, func(ctx context.Context) error {
				if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
				values ($1, $2, $3) on conflict (id) do update set value = $3`, v.ID, v.MType, v.Value); err != nil {
					return retry.RetryableError(fmt.Errorf("write gauge to db failed: %s", err.Error()))
				}
				return nil
			}); err != nil {
				return err
			}
		case metrictypes.CounterType:
			if v.Delta == nil || v.ID == "" {
				log.Println("empty id or nil value counter metric")
				continue
			}
			if err := retry.Do(ctx, bf, func(ctx context.Context) error {
				if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
				values ($1, $2, $3) on conflict (id) 
				do update set delta = $3 + (select delta from monitoring where id = $1)`, v.ID, v.MType, v.Delta); err != nil {
					return retry.RetryableError(fmt.Errorf("write counter to db failed: %s", err.Error()))
				}
				return nil
			}); err != nil {
				return err
			}
		default:
			log.Println("wrong metric type")
			continue
		}
	}
	return tx.Commit()
}
func (p *PgDB) GetAllMetricsTxt() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	bf := retry.WithMaxRetries(p.backoff.maxRetries, retry.NewFibonacci(p.backoff.fiboDuration))

	s := "---Counters---\n"
	var id string
	var delta int64
	var value float64
	var rowsc, rowsg *sql.Rows
	var err error

	if err = retry.Do(ctx, bf, func(ctx context.Context) error {
		rowsc, err = p.db.QueryContext(ctx, "select id, delta from monitoring where mtype = 'counter'")
		if err != nil {
			return retry.RetryableError(err)
		}
		//static check lying
		if err = rowsc.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	defer rowsc.Close()
	for rowsc.Next() {
		if err := rowsc.Scan(&id, &delta); err != nil {
			return "", err
		}
		s += fmt.Sprintf("%v: %v\n", id, delta)
	}
	if err := rowsc.Err(); err != nil {
		return "", err
	}
	s += "---Gauge---\n"
	if err = retry.Do(ctx, bf, func(ctx context.Context) error {
		rowsg, err = p.db.QueryContext(ctx, "select id, value from monitoring where mtype = 'gauge'")
		if err != nil {
			return retry.RetryableError(err)
		}
		//static check lying
		if err = rowsg.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	defer rowsg.Close()
	for rowsg.Next() {
		if err := rowsg.Scan(&id, &value); err != nil {
			return "", err
		}
		s += fmt.Sprintf("%v: %v\n", id, value)
	}
	if err := rowsg.Err(); err != nil {
		return "", err
	}

	return s, nil
}
func (p *PgDB) GetMetric(mType, name string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	bf := retry.WithMaxRetries(p.backoff.maxRetries, retry.NewFibonacci(p.backoff.fiboDuration))

	var value float64
	var delta int64
	if mType == metrictypes.GaugeType {
		if err := retry.Do(ctx, bf, func(ctx context.Context) error {
			row := p.db.QueryRowContext(ctx, "select value from monitoring where id = $1", name)
			if err := row.Scan(&value); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return errors.New("no value")
				}
				return retry.RetryableError(err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return metrictypes.Gauge(value), nil
	} else if mType == metrictypes.CounterType {
		if err := retry.Do(ctx, bf, func(ctx context.Context) error {
			row := p.db.QueryRowContext(ctx, "select delta from monitoring where id = $1", name)
			if err := row.Scan(&delta); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return errors.New("no value")
				}
				return retry.RetryableError(err)
			}
			return nil
		}); err != nil {
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
func (p *PgDB) SetBackoff(fibotime time.Duration, maxretries uint64) {
	p.backoff.fiboDuration = fibotime
	p.backoff.maxRetries = maxretries
}
