package customerrors

import "errors"

var (
	ErrNoVal                = errors.New("no value")
	ErrBadMetricType        = errors.New("bad metric type")
	ErrWrongMetricType      = errors.New("wrong metric type")
	ErrWrongMetricValueType = errors.New("wrong metric value type")
)
