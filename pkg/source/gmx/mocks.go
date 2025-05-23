// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx (interfaces: IVaultReader,IVaultPriceFeedReader,IFastPriceFeedV1Reader,IPriceFeedReader,IUSDGReader,IChainlinkFlagsReader,IPancakePairReader)
//
// Generated by this command:
//
//	mockgen -destination ./mocks.go -package gmx . IVaultReader,IVaultPriceFeedReader,IFastPriceFeedV1Reader,IPriceFeedReader,IUSDGReader,IChainlinkFlagsReader,IPancakePairReader
//

// Package gmx is a generated GoMock package.
package gmx

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockIVaultReader is a mock of IVaultReader interface.
type MockIVaultReader struct {
	ctrl     *gomock.Controller
	recorder *MockIVaultReaderMockRecorder
	isgomock struct{}
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
func (m *MockIVaultReader) Read(ctx context.Context, address string) (*Vault, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address)
	ret0, _ := ret[0].(*Vault)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIVaultReaderMockRecorder) Read(ctx, address any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIVaultReader)(nil).Read), ctx, address)
}

// MockIVaultPriceFeedReader is a mock of IVaultPriceFeedReader interface.
type MockIVaultPriceFeedReader struct {
	ctrl     *gomock.Controller
	recorder *MockIVaultPriceFeedReaderMockRecorder
	isgomock struct{}
}

// MockIVaultPriceFeedReaderMockRecorder is the mock recorder for MockIVaultPriceFeedReader.
type MockIVaultPriceFeedReaderMockRecorder struct {
	mock *MockIVaultPriceFeedReader
}

// NewMockIVaultPriceFeedReader creates a new mock instance.
func NewMockIVaultPriceFeedReader(ctrl *gomock.Controller) *MockIVaultPriceFeedReader {
	mock := &MockIVaultPriceFeedReader{ctrl: ctrl}
	mock.recorder = &MockIVaultPriceFeedReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIVaultPriceFeedReader) EXPECT() *MockIVaultPriceFeedReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIVaultPriceFeedReader) Read(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address, tokens)
	ret0, _ := ret[0].(*VaultPriceFeed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIVaultPriceFeedReaderMockRecorder) Read(ctx, address, tokens any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIVaultPriceFeedReader)(nil).Read), ctx, address, tokens)
}

// MockIFastPriceFeedV1Reader is a mock of IFastPriceFeedV1Reader interface.
type MockIFastPriceFeedV1Reader struct {
	ctrl     *gomock.Controller
	recorder *MockIFastPriceFeedV1ReaderMockRecorder
	isgomock struct{}
}

// MockIFastPriceFeedV1ReaderMockRecorder is the mock recorder for MockIFastPriceFeedV1Reader.
type MockIFastPriceFeedV1ReaderMockRecorder struct {
	mock *MockIFastPriceFeedV1Reader
}

// NewMockIFastPriceFeedV1Reader creates a new mock instance.
func NewMockIFastPriceFeedV1Reader(ctrl *gomock.Controller) *MockIFastPriceFeedV1Reader {
	mock := &MockIFastPriceFeedV1Reader{ctrl: ctrl}
	mock.recorder = &MockIFastPriceFeedV1ReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIFastPriceFeedV1Reader) EXPECT() *MockIFastPriceFeedV1ReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIFastPriceFeedV1Reader) Read(ctx context.Context, address string, tokens []string) (*FastPriceFeedV1, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address, tokens)
	ret0, _ := ret[0].(*FastPriceFeedV1)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIFastPriceFeedV1ReaderMockRecorder) Read(ctx, address, tokens any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIFastPriceFeedV1Reader)(nil).Read), ctx, address, tokens)
}

// MockIPriceFeedReader is a mock of IPriceFeedReader interface.
type MockIPriceFeedReader struct {
	ctrl     *gomock.Controller
	recorder *MockIPriceFeedReaderMockRecorder
	isgomock struct{}
}

// MockIPriceFeedReaderMockRecorder is the mock recorder for MockIPriceFeedReader.
type MockIPriceFeedReaderMockRecorder struct {
	mock *MockIPriceFeedReader
}

// NewMockIPriceFeedReader creates a new mock instance.
func NewMockIPriceFeedReader(ctrl *gomock.Controller) *MockIPriceFeedReader {
	mock := &MockIPriceFeedReader{ctrl: ctrl}
	mock.recorder = &MockIPriceFeedReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPriceFeedReader) EXPECT() *MockIPriceFeedReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIPriceFeedReader) Read(ctx context.Context, address string, roundCount int) (*PriceFeed, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address, roundCount)
	ret0, _ := ret[0].(*PriceFeed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIPriceFeedReaderMockRecorder) Read(ctx, address, roundCount any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIPriceFeedReader)(nil).Read), ctx, address, roundCount)
}

// MockIUSDGReader is a mock of IUSDGReader interface.
type MockIUSDGReader struct {
	ctrl     *gomock.Controller
	recorder *MockIUSDGReaderMockRecorder
	isgomock struct{}
}

// MockIUSDGReaderMockRecorder is the mock recorder for MockIUSDGReader.
type MockIUSDGReaderMockRecorder struct {
	mock *MockIUSDGReader
}

// NewMockIUSDGReader creates a new mock instance.
func NewMockIUSDGReader(ctrl *gomock.Controller) *MockIUSDGReader {
	mock := &MockIUSDGReader{ctrl: ctrl}
	mock.recorder = &MockIUSDGReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIUSDGReader) EXPECT() *MockIUSDGReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIUSDGReader) Read(ctx context.Context, address string) (*USDG, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address)
	ret0, _ := ret[0].(*USDG)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIUSDGReaderMockRecorder) Read(ctx, address any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIUSDGReader)(nil).Read), ctx, address)
}

// MockIChainlinkFlagsReader is a mock of IChainlinkFlagsReader interface.
type MockIChainlinkFlagsReader struct {
	ctrl     *gomock.Controller
	recorder *MockIChainlinkFlagsReaderMockRecorder
	isgomock struct{}
}

// MockIChainlinkFlagsReaderMockRecorder is the mock recorder for MockIChainlinkFlagsReader.
type MockIChainlinkFlagsReaderMockRecorder struct {
	mock *MockIChainlinkFlagsReader
}

// NewMockIChainlinkFlagsReader creates a new mock instance.
func NewMockIChainlinkFlagsReader(ctrl *gomock.Controller) *MockIChainlinkFlagsReader {
	mock := &MockIChainlinkFlagsReader{ctrl: ctrl}
	mock.recorder = &MockIChainlinkFlagsReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIChainlinkFlagsReader) EXPECT() *MockIChainlinkFlagsReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIChainlinkFlagsReader) Read(ctx context.Context, address string) (*ChainlinkFlags, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address)
	ret0, _ := ret[0].(*ChainlinkFlags)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIChainlinkFlagsReaderMockRecorder) Read(ctx, address any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIChainlinkFlagsReader)(nil).Read), ctx, address)
}

// MockIPancakePairReader is a mock of IPancakePairReader interface.
type MockIPancakePairReader struct {
	ctrl     *gomock.Controller
	recorder *MockIPancakePairReaderMockRecorder
	isgomock struct{}
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
func (m *MockIPancakePairReader) Read(ctx context.Context, address string) (*PancakePair, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, address)
	ret0, _ := ret[0].(*PancakePair)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIPancakePairReaderMockRecorder) Read(ctx, address any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIPancakePairReader)(nil).Read), ctx, address)
}
