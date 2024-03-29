// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx (interfaces: IPancakePairReader)

// Package gmx is a generated GoMock package.
package quickperps

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockIPancakePairReader is a mock of IPancakePairReader interface.
type MockIPancakePairReader struct {
	ctrl     *gomock.Controller
	recorder *MockIPancakePairReaderMockRecorder
}

// MockIPancakePairReaderMockRecorder is the mock recorder for MockIPancakePairReader.
type MockIPancakePairReaderMockRecorder struct {
	mock *MockIPancakePairReader
}

// NewMockIPancakePairReader creates a new mock instance.
func NewMockIPancakePairReader(ctrl *gomock.Controller) *MockIPancakePairReader {
	mock := &MockIPancakePairReader{ctrl: ctrl}
	mock.recorder = &MockIPancakePairReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPancakePairReader) EXPECT() *MockIPancakePairReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIPancakePairReader) Read(arg0 context.Context, arg1 string) (*PancakePair, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", arg0, arg1)
	ret0, _ := ret[0].(*PancakePair)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIPancakePairReaderMockRecorder) Read(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIPancakePairReader)(nil).Read), arg0, arg1)
}
