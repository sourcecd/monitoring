package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sourcecd/monitoring/internal/customerrors"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

const populateQuery = `create table if not exists monitoring ( id varchar(64) PRIMARY KEY, 
mtype varchar(16), delta bigint, value double precision )`

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

func (p *PgDB) PopulateDB(ctx context.Context) error {
	if _, err := p.db.ExecContext(ctx, populateQuery); err != nil {
		return fmt.Errorf("populate failed: %s", err.Error())
	}
	return nil
}

func (p *PgDB) WriteMetric(ctx context.Context, mtype, name string, val interface{}) error {
	switch mtype {
	case metrictypes.GaugeType:
		if metric, ok := val.(metrictypes.Gauge); ok {
			//idempotency
			if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
			values ($1, $2, $3) on conflict (id) do update set value = $3`, name, "gauge", metric); err != nil {
				return fmt.Errorf("write gauge to db failed: %s", err.Error())
			}
			return nil
		}
		return errors.New("wrong metric value type")
	case metrictypes.CounterType:
		if metric, ok := val.(metrictypes.Counter); ok {
			if _, err := p.db.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
			values ($1, $2, $3) on conflict (id) 
			do update set delta = $3 + (select delta from monitoring where id = $1)`, name, "counter", metric); err != nil {
				return fmt.Errorf("write counter to db failed: %s", err.Error())
			}
			return nil
		}
		return customerrors.ErrWrongMetricValueType
	default:
		return customerrors.ErrWrongMetricType
	}
}
func (p *PgDB) WriteBatchMetrics(ctx context.Context, metrics []models.Metrics) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("can't start tx to db: %s", err.Error())
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
			if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, value) 
			values ($1, $2, $3) on conflict (id) do update set value = $3`, v.ID, v.MType, v.Value); err != nil {
				return fmt.Errorf("write gauge to db failed: %s", err.Error())
			}
		case metrictypes.CounterType:
			if v.Delta == nil || v.ID == "" {
				log.Println("empty id or nil value counter metric")
				continue
			}
			if _, err := tx.ExecContext(ctx, `insert into monitoring (id, mtype, delta) 
			values ($1, $2, $3) on conflict (id) 
			do update set delta = $3 + (select delta from monitoring where id = $1)`, v.ID, v.MType, v.Delta); err != nil {
				return fmt.Errorf("write counter to db failed: %s", err.Error())
			}
		default:
			log.Println("wrong metric type")
			continue
		}
	}
	return tx.Commit()
}
func (p *PgDB) GetAllMetricsTxt(ctx context.Context) (string, error) {
	s := "---Counters---\n"
	var id string
	var delta int64
	var value float64

	rowsc, err := p.db.QueryContext(ctx, "select id, delta from monitoring where mtype = 'counter' order by id")
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
	if err := rowsc.Err(); err != nil {
		return "", err
	}
	s += "---Gauge---\n"
	rowsg, err := p.db.QueryContext(ctx, "select id, value from monitoring where mtype = 'gauge' order by id")
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
	if err := rowsg.Err(); err != nil {
		return "", err
	}

	return s, nil
}
func (p *PgDB) GetMetric(ctx context.Context, mType, name string) (interface{}, error) {
	var value float64
	var delta int64
	if mType == metrictypes.GaugeType {
		row := p.db.QueryRowContext(ctx, "select value from monitoring where id = $1", name)
		if err := row.Scan(&value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, customerrors.ErrNoVal
			}
			return nil, err
		}
		return metrictypes.Gauge(value), nil
	} else if mType == metrictypes.CounterType {
		row := p.db.QueryRowContext(ctx, "select delta from monitoring where id = $1", name)
		if err := row.Scan(&delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, customerrors.ErrNoVal
			}
			return nil, err
		}
		return metrictypes.Counter(delta), nil
	} else {
		return nil, customerrors.ErrBadMetricType
	}
}

func (p *PgDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
func (p *PgDB) CloseDB() error {
	return p.db.Close()
}
