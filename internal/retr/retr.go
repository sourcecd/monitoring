package retr

import (
	"context"
	"errors"
	"time"

	"github.com/sethvargo/go-retry"

	"github.com/sourcecd/monitoring/internal/customerrors"
	"github.com/sourcecd/monitoring/internal/models"
)

type (
	Retr struct {
		maxRetries uint64
		fiboDuration,
		timeout time.Duration
		skippedErrors error
	}

	WriteMetricType       func(ctx context.Context, mtype, name string, val interface{}) error
	WriteBatchMetricsType func(ctx context.Context, metrics []models.Metrics) error
	PopulateDBType        func(ctx context.Context) error
	GetAllMetricsTxtType  func(ctx context.Context) (string, error)
	GetMetricType         func(ctx context.Context, mType, name string) (interface{}, error)
)

func (rtr *Retr) UseRetrWM(f WriteMetricType) WriteMetricType {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, mtype, name string, val interface{}) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, mtype, name, val)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) UseRetrWMB(f WriteBatchMetricsType) WriteBatchMetricsType {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, metrics []models.Metrics) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, metrics)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) UseRetrPopDB(f PopulateDBType) PopulateDBType {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) UseRetrGetAllM(f GetAllMetricsTxtType) GetAllMetricsTxtType {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		var s string
		var err error
		err = retry.Do(ctx, bf, func(ctx context.Context) error {
			s, err = f(ctx)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return s, err
	}
}

func (rtr *Retr) UseRetrGetMetric(f GetMetricType) GetMetricType {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, mType, name string) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		var i interface{}
		var err error
		err = retry.Do(ctx, bf, func(ctx context.Context) error {
			i, err = f(ctx, mType, name)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return i, err
	}
}

func (rtr *Retr) SetParams(fibotime, timeout time.Duration, maxretries uint64) {
	rtr.fiboDuration = fibotime
	rtr.maxRetries = maxretries
	rtr.timeout = timeout
}

func NewRetr() *Retr {
	return &Retr{
		fiboDuration: 1 * time.Second,
		maxRetries:   3,
		timeout:      30 * time.Second,
		skippedErrors: errors.Join(
			customerrors.ErrWrongMetricValueType,
			customerrors.ErrWrongMetricType,
			customerrors.ErrBadMetricType,
			customerrors.ErrNoVal,
		),
	}
}

func (rtr *Retr) GetTimeoutCtx() time.Duration {
	return rtr.timeout
}
