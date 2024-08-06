// Package retrier retry logic for monitoring methods.
package retrier

import (
	"context"
	"errors"
	"time"

	"github.com/sethvargo/go-retry"

	"github.com/sourcecd/monitoring/internal/customerrors"
	"github.com/sourcecd/monitoring/internal/models"
)

type (
	// Retrier type of retry subsystem.
	Retrier struct {
		skippedErrors error         // non-retriable errors
		maxRetries    uint64        // maximum retry counts
		fiboDuration  time.Duration // duration between retries by fibonacci algoritm
		timeout       time.Duration // retry timeout
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

// UseRetrierWM retry method for WriteMetric function.
func (reqRetrier *Retrier) UseRetrierWM(f WriteMetricType) WriteMetricType {
	bf := retry.WithMaxRetries(reqRetrier.maxRetries, retry.NewFibonacci(reqRetrier.fiboDuration))

	return func(ctx context.Context, mtype, name string, val interface{}) error {
		ctx, cancel := context.WithTimeout(ctx, reqRetrier.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, mtype, name, val)
			if errors.Is(reqRetrier.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

// UseRetrierWMB retry method for WriteBatchMetrics function.
func (reqRetrier *Retrier) UseRetrierWMB(f WriteBatchMetricsType) WriteBatchMetricsType {
	bf := retry.WithMaxRetries(reqRetrier.maxRetries, retry.NewFibonacci(reqRetrier.fiboDuration))

	return func(ctx context.Context, metrics []models.Metrics) error {
		ctx, cancel := context.WithTimeout(ctx, reqRetrier.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, metrics)
			if errors.Is(reqRetrier.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

// UseRetrierPopDB retry method for PopulateDB function.
func (reqRetrier *Retrier) UseRetrierPopDB(f PopulateDBType) PopulateDBType {
	bf := retry.WithMaxRetries(reqRetrier.maxRetries, retry.NewFibonacci(reqRetrier.fiboDuration))

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, reqRetrier.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx)
			if errors.Is(reqRetrier.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

// UseRetrierGetAllM retry method for GetAllMetricsTxt function.
func (reqRetrier *Retrier) UseRetrierGetAllM(f GetAllMetricsTxtType) GetAllMetricsTxtType {
	bf := retry.WithMaxRetries(reqRetrier.maxRetries, retry.NewFibonacci(reqRetrier.fiboDuration))

	return func(ctx context.Context) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, reqRetrier.timeout)
		defer cancel()
		var s string
		var err error
		err = retry.Do(ctx, bf, func(ctx context.Context) error {
			s, err = f(ctx)
			if errors.Is(reqRetrier.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return s, err
	}
}

// UseRetrierGetMetric retry method for GetMetric function.
func (reqRetrier *Retrier) UseRetrierGetMetric(f GetMetricType) GetMetricType {
	bf := retry.WithMaxRetries(reqRetrier.maxRetries, retry.NewFibonacci(reqRetrier.fiboDuration))

	return func(ctx context.Context, mType, name string) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, reqRetrier.timeout)
		defer cancel()
		var i interface{}
		var err error
		err = retry.Do(ctx, bf, func(ctx context.Context) error {
			i, err = f(ctx, mType, name)
			if errors.Is(reqRetrier.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return i, err
	}
}

// SetParams set retry parameters.
func (reqRetrier *Retrier) SetParams(fibotime, timeout time.Duration, maxretries uint64) {
	reqRetrier.fiboDuration = fibotime
	reqRetrier.maxRetries = maxretries
	reqRetrier.timeout = timeout
}

// NewRetrier init retrier.
func NewRetrier() *Retrier {
	return &Retrier{
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
func (reqRetrier *Retrier) GetTimeoutCtx() time.Duration {
	return reqRetrier.timeout
}
