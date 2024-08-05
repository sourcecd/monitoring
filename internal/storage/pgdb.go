// Package storage postgress implementation
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

// Sql query for create monitoring table in postgres DB.
const populateQuery = `create table if not exists monitoring ( id varchar(64) PRIMARY KEY, 
mtype varchar(16), delta bigint, value double precision )`

// PgDB singleton type for connect and work with postgres DB.
type PgDB struct {
	db                *sql.DB
	getGaugeStmt      *sql.Stmt
	getCounterStmt    *sql.Stmt
	getAllGaugeStmt   *sql.Stmt
	getAllCounterStmt *sql.Stmt
	insertGaugeStmt   *sql.Stmt
	insertCounterStmt *sql.Stmt
}

// Prepare queries
func (p *PgDB) prepareStatements() error {
	var err error
	p.getGaugeStmt, err = p.db.Prepare("SELECT value FROM monitoring WHERE id = $1")
	if err != nil {
		return err
	}
	p.getCounterStmt, err = p.db.Prepare("SELECT delta FROM monitoring WHERE id = $1")
	if err != nil {
		return err
	}
	p.getAllGaugeStmt, err = p.db.Prepare("SELECT id, value FROM monitoring WHERE mtype = 'gauge' ORDER BY id")
	if err != nil {
		return err
	}
	p.getAllCounterStmt, err = p.db.Prepare("SELECT id, delta FROM monitoring WHERE mtype = 'counter' ORDER BY id")
	if err != nil {
		return err
	}
	p.insertGaugeStmt, err = p.db.Prepare("INSERT INTO monitoring (id, mtype, value) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET value = $3")
	if err != nil {
		return err
	}
	p.insertCounterStmt, err = p.db.Prepare("INSERT INTO monitoring (id, mtype, delta) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET delta = $3 + (SELECT delta FROM monitoring WHERE id = $1)")
	return err
}

// NewPgDB init postgres DB.
func NewPgDB(dsn string) (*PgDB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PgDB{db: db}, nil
}

// PopulateDB method for create monitoring table.
func (p *PgDB) PopulateDB(ctx context.Context) error {
	if _, err := p.db.ExecContext(ctx, populateQuery); err != nil {
		return fmt.Errorf("populate failed: %s", err.Error())
	}
	if err := p.prepareStatements(); err != nil {
		return err
	}
	return nil
}

// WriteMetric implementation WriteMetric method of storage interface (postgres DB storage).
func (p *PgDB) WriteMetric(ctx context.Context, mtype, name string, val interface{}) error {
	// selecting metric type
	switch mtype {
	case metrictypes.GaugeType:
		if metric, ok := val.(metrictypes.Gauge); ok {
			//idempotency
			if _, err := p.insertGaugeStmt.ExecContext(ctx, name, "gauge", metric); err != nil {
				return fmt.Errorf("write gauge to db failed: %s", err.Error())
			}
			return nil
		}
		return customerrors.ErrWrongMetricValueType
	case metrictypes.CounterType:
		if metric, ok := val.(metrictypes.Counter); ok {
			if _, err := p.insertCounterStmt.ExecContext(ctx, name, "counter", metric); err != nil {
				return fmt.Errorf("write counter to db failed: %s", err.Error())
			}
			return nil
		}
		return customerrors.ErrWrongMetricValueType
	default:
		return customerrors.ErrWrongMetricType
	}
}

// WriteBatchMetrics implementation WriteBatchMetrics method of storage interface (postgres DB storage).
func (p *PgDB) WriteBatchMetrics(ctx context.Context, metrics []models.Metrics) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("can't start tx to db: %s", err.Error())
	}
	defer tx.Rollback()

	// i think we don't break all batch if one metric failed in batch (use continue)
	for _, v := range metrics {
		// selecting metric type
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

// GetAllMetricsTxt implementation GetAllMetricsTxt method of storage interface (postgres DB storage).
func (p *PgDB) GetAllMetricsTxt(ctx context.Context) (string, error) {
	s := "---Counters---\n"
	var id string
	var delta int64
	var value float64

	rowsc, err := p.getAllCounterStmt.QueryContext(ctx)
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
	rowsg, err := p.getAllGaugeStmt.QueryContext(ctx)
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

// GetMetric implementation GetMetric method of storage interface (postgres DB storage).
func (p *PgDB) GetMetric(ctx context.Context, mType, name string) (interface{}, error) {
	var value float64
	var delta int64
	// selecting metric type
	switch mType {
	case metrictypes.GaugeType:
		row := p.getGaugeStmt.QueryRowContext(ctx, name)
		if err := row.Scan(&value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, customerrors.ErrNoVal
			}
			return nil, err
		}
		return metrictypes.Gauge(value), nil
	case metrictypes.CounterType:
		row := p.getCounterStmt.QueryRowContext(ctx, name)
		if err := row.Scan(&delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, customerrors.ErrNoVal
			}
			return nil, err
		}
		return metrictypes.Counter(delta), nil
	default:
		return nil, customerrors.ErrBadMetricType
	}
}

// Ping implementation Ping method of storage interface (postgres DB storage).
func (p *PgDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// CloseDB close connection to database.
func (p *PgDB) CloseDB() error {
	return p.db.Close()
}
