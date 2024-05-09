// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/sourcecd/monitoring/internal/storage (interfaces: StoreMetrics)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/sourcecd/monitoring/internal/models"
)

// MockStoreMetrics is a mock of StoreMetrics interface.
type MockStoreMetrics struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMetricsMockRecorder
}

// MockStoreMetricsMockRecorder is the mock recorder for MockStoreMetrics.
type MockStoreMetricsMockRecorder struct {
	mock *MockStoreMetrics
}

// NewMockStoreMetrics creates a new mock instance.
func NewMockStoreMetrics(ctrl *gomock.Controller) *MockStoreMetrics {
	mock := &MockStoreMetrics{ctrl: ctrl}
	mock.recorder = &MockStoreMetricsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStoreMetrics) EXPECT() *MockStoreMetricsMockRecorder {
	return m.recorder
}

// GetAllMetricsTxt mocks base method.
func (m *MockStoreMetrics) GetAllMetricsTxt() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllMetricsTxt")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllMetricsTxt indicates an expected call of GetAllMetricsTxt.
func (mr *MockStoreMetricsMockRecorder) GetAllMetricsTxt() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllMetricsTxt", reflect.TypeOf((*MockStoreMetrics)(nil).GetAllMetricsTxt))
}

// GetMetric mocks base method.
func (m *MockStoreMetrics) GetMetric(arg0, arg1 string) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMetric", arg0, arg1)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMetric indicates an expected call of GetMetric.
func (mr *MockStoreMetricsMockRecorder) GetMetric(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMetric", reflect.TypeOf((*MockStoreMetrics)(nil).GetMetric), arg0, arg1)
}

// Ping mocks base method.
func (m *MockStoreMetrics) Ping() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ping")
	ret0, _ := ret[0].(error)
	return ret0
}

// Ping indicates an expected call of Ping.
func (mr *MockStoreMetricsMockRecorder) Ping() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ping", reflect.TypeOf((*MockStoreMetrics)(nil).Ping))
}

// WriteBatchMetrics mocks base method.
func (m *MockStoreMetrics) WriteBatchMetrics(arg0 []models.Metrics) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteBatchMetrics", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteBatchMetrics indicates an expected call of WriteBatchMetrics.
func (mr *MockStoreMetricsMockRecorder) WriteBatchMetrics(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteBatchMetrics", reflect.TypeOf((*MockStoreMetrics)(nil).WriteBatchMetrics), arg0)
}

// WriteMetric mocks base method.
func (m *MockStoreMetrics) WriteMetric(arg0, arg1 string, arg2 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteMetric", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteMetric indicates an expected call of WriteMetric.
func (mr *MockStoreMetricsMockRecorder) WriteMetric(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteMetric", reflect.TypeOf((*MockStoreMetrics)(nil).WriteMetric), arg0, arg1, arg2)
}
