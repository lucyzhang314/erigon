// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/erigontech/erigon/cl/phase1/network/services (interfaces: AggregateAndProofService)
//
// Generated by this command:
//
//	mockgen -typed=true -destination=./mock_services/aggregate_and_proof_service_mock.go -package=mock_services . AggregateAndProofService
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	cltypes "github.com/erigontech/erigon/cl/cltypes"
	gomock "go.uber.org/mock/gomock"
)

// MockAggregateAndProofService is a mock of AggregateAndProofService interface.
type MockAggregateAndProofService struct {
	ctrl     *gomock.Controller
	recorder *MockAggregateAndProofServiceMockRecorder
	isgomock struct{}
}

// MockAggregateAndProofServiceMockRecorder is the mock recorder for MockAggregateAndProofService.
type MockAggregateAndProofServiceMockRecorder struct {
	mock *MockAggregateAndProofService
}

// NewMockAggregateAndProofService creates a new mock instance.
func NewMockAggregateAndProofService(ctrl *gomock.Controller) *MockAggregateAndProofService {
	mock := &MockAggregateAndProofService{ctrl: ctrl}
	mock.recorder = &MockAggregateAndProofServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAggregateAndProofService) EXPECT() *MockAggregateAndProofServiceMockRecorder {
	return m.recorder
}

// ProcessMessage mocks base method.
func (m *MockAggregateAndProofService) ProcessMessage(ctx context.Context, subnet *uint64, msg *cltypes.SignedAggregateAndProofData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessMessage", ctx, subnet, msg)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessMessage indicates an expected call of ProcessMessage.
func (mr *MockAggregateAndProofServiceMockRecorder) ProcessMessage(ctx, subnet, msg any) *MockAggregateAndProofServiceProcessMessageCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessMessage", reflect.TypeOf((*MockAggregateAndProofService)(nil).ProcessMessage), ctx, subnet, msg)
	return &MockAggregateAndProofServiceProcessMessageCall{Call: call}
}

// MockAggregateAndProofServiceProcessMessageCall wrap *gomock.Call
type MockAggregateAndProofServiceProcessMessageCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAggregateAndProofServiceProcessMessageCall) Return(arg0 error) *MockAggregateAndProofServiceProcessMessageCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAggregateAndProofServiceProcessMessageCall) Do(f func(context.Context, *uint64, *cltypes.SignedAggregateAndProofData) error) *MockAggregateAndProofServiceProcessMessageCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAggregateAndProofServiceProcessMessageCall) DoAndReturn(f func(context.Context, *uint64, *cltypes.SignedAggregateAndProofData) error) *MockAggregateAndProofServiceProcessMessageCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
