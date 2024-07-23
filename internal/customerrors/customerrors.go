// Service specified error types.
package customerrors

import "errors"

var (
	ErrNoVal                = errors.New("no value")                // error type for "no value"
	ErrBadMetricType        = errors.New("bad metric type")         // error type for incorrect metric type (Get)
	ErrWrongMetricType      = errors.New("wrong metric type")       // error type for incorrect metric type (Write)
	ErrWrongMetricValueType = errors.New("wrong metric value type") // error type for incorrect value of specified metric type
)
