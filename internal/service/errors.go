package service

import "errors"

var (
	ErrMetricNotFound  = errors.New("metric not found")
	ErrUnknownType     = errors.New("unknown metric type")
	ErrMetricNameEmpty = errors.New("metric name is empty")
	ErrValueRequired   = errors.New("value is required for gauge")
	ErrDeltaRequired   = errors.New("delta is required for counter")
)
