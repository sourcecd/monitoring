// Package retr retry logic for monitoring methods.
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
	// Retr type of retry subsystem.
	Retr struct {
		maxRetries    uint64        // maximum retry counts
		fiboDuration  time.Duration // duration between retries by fibonacci algoritm
		timeout       time.Duration // retry timeout
		skippedErrors error         // non-retriable errors
	}

	// WriteMetricType type of function for WriteMetricType method retry.
	WriteMetricType func(ctx context.Context, mtype, name string, val interface{}) error
	// WriteBatchMetricsType type of function for WriteBatchMetricsType method retry.
	WriteBatchMetricsType func(ctx context.Context, metrics []models.Metrics) error
	// PopulateDBType type of function for PopulateDBType method retry.
	PopulateDBType func(ctx context.Context) error
	// GetAllMetricsTxtType type of function for GetAllMetricsTxtType method retry.
	GetAllMetricsTxtType func(ctx context.Context) (string, error)
	// GetMetricType type of function for GetMetricType method retry.
	GetMetricType func(ctx context.Context, mType, name string) (interface{}, error)
)

// UseRetrWM retry method for WriteMetric function.
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

// UseRetrWMB retry method for WriteBatchMetrics function.
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

// UseRetrPopDB retry method for PopulateDB function.
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

// UseRetrGetAllM retry method for GetAllMetricsTxt function.
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

// UseRetrGetMetric retry method for GetMetric function.
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

// SetParams set retry parameters.
func (rtr *Retr) SetParams(fibotime, timeout time.Duration, maxretries uint64) {
	rtr.fiboDuration = fibotime
	rtr.maxRetries = maxretries
	rtr.timeout = timeout
}

// NewRetr init retrier.
func NewRetr() *Retr {
	return &Retr{
		fiboDuration: 1 * time.Second,
		maxRetries:   3,
		timeout:      30 * time.Second,
		// non-retrieble errors
		skippedErrors: errors.Join(
			customerrors.ErrWrongMetricValueType,
			customerrors.ErrWrongMetricType,
			customerrors.ErrBadMetricType,
			customerrors.ErrNoVal,
		),
	}
}

// GetTimeoutCtx supportive method for get current timeout setting.
func (rtr *Retr) GetTimeoutCtx() time.Duration {
	return rtr.timeout
}
