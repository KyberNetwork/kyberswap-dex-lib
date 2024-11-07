// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx (interfaces: IVaultReader)

// Package gmx is a generated GoMock package.
package quickperps

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockIVaultReader is a mock of IVaultReader interface.
type MockIVaultReader struct {
	ctrl     *gomock.Controller
	recorder *MockIVaultReaderMockRecorder
}

// MockIVaultReaderMockRecorder is the mock recorder for MockIVaultReader.
type MockIVaultReaderMockRecorder struct {
	mock *MockIVaultReader
}

// NewMockIVaultReader creates a new mock instance.
func NewMockIVaultReader(ctrl *gomock.Controller) *MockIVaultReader {
	mock := &MockIVaultReader{ctrl: ctrl}
	mock.recorder = &MockIVaultReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIVaultReader) EXPECT() *MockIVaultReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIVaultReader) Read(arg0 context.Context, arg1 string) (*Vault, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", arg0, arg1)
	ret0, _ := ret[0].(*Vault)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIVaultReaderMockRecorder) Read(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIVaultReader)(nil).Read), arg0, arg1)
}
