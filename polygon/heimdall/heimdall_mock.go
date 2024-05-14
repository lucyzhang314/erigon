// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ledgerwatch/erigon/polygon/heimdall (interfaces: Heimdall)
//
// Generated by this command:
//
//	mockgen -typed=true -destination=./heimdall_mock.go -package=heimdall . Heimdall
//

// Package heimdall is a generated GoMock package.
package heimdall

import (
	context "context"
	reflect "reflect"

	polygoncommon "github.com/ledgerwatch/erigon/polygon/polygoncommon"
	gomock "go.uber.org/mock/gomock"
)

// MockHeimdall is a mock of Heimdall interface.
type MockHeimdall struct {
	ctrl     *gomock.Controller
	recorder *MockHeimdallMockRecorder
}

// MockHeimdallMockRecorder is the mock recorder for MockHeimdall.
type MockHeimdallMockRecorder struct {
	mock *MockHeimdall
}

// NewMockHeimdall creates a new mock instance.
func NewMockHeimdall(ctrl *gomock.Controller) *MockHeimdall {
	mock := &MockHeimdall{ctrl: ctrl}
	mock.recorder = &MockHeimdallMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHeimdall) EXPECT() *MockHeimdallMockRecorder {
	return m.recorder
}

// FetchCheckpointsFromBlock mocks base method.
func (m *MockHeimdall) FetchCheckpointsFromBlock(arg0 context.Context, arg1 uint64) (Waypoints, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchCheckpointsFromBlock", arg0, arg1)
	ret0, _ := ret[0].(Waypoints)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchCheckpointsFromBlock indicates an expected call of FetchCheckpointsFromBlock.
func (mr *MockHeimdallMockRecorder) FetchCheckpointsFromBlock(arg0, arg1 any) *MockHeimdallFetchCheckpointsFromBlockCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchCheckpointsFromBlock", reflect.TypeOf((*MockHeimdall)(nil).FetchCheckpointsFromBlock), arg0, arg1)
	return &MockHeimdallFetchCheckpointsFromBlockCall{Call: call}
}

// MockHeimdallFetchCheckpointsFromBlockCall wrap *gomock.Call
type MockHeimdallFetchCheckpointsFromBlockCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHeimdallFetchCheckpointsFromBlockCall) Return(arg0 Waypoints, arg1 error) *MockHeimdallFetchCheckpointsFromBlockCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHeimdallFetchCheckpointsFromBlockCall) Do(f func(context.Context, uint64) (Waypoints, error)) *MockHeimdallFetchCheckpointsFromBlockCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHeimdallFetchCheckpointsFromBlockCall) DoAndReturn(f func(context.Context, uint64) (Waypoints, error)) *MockHeimdallFetchCheckpointsFromBlockCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// FetchLatestSpan mocks base method.
func (m *MockHeimdall) FetchLatestSpan(arg0 context.Context) (*Span, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchLatestSpan", arg0)
	ret0, _ := ret[0].(*Span)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchLatestSpan indicates an expected call of FetchLatestSpan.
func (mr *MockHeimdallMockRecorder) FetchLatestSpan(arg0 any) *MockHeimdallFetchLatestSpanCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchLatestSpan", reflect.TypeOf((*MockHeimdall)(nil).FetchLatestSpan), arg0)
	return &MockHeimdallFetchLatestSpanCall{Call: call}
}

// MockHeimdallFetchLatestSpanCall wrap *gomock.Call
type MockHeimdallFetchLatestSpanCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHeimdallFetchLatestSpanCall) Return(arg0 *Span, arg1 error) *MockHeimdallFetchLatestSpanCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHeimdallFetchLatestSpanCall) Do(f func(context.Context) (*Span, error)) *MockHeimdallFetchLatestSpanCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHeimdallFetchLatestSpanCall) DoAndReturn(f func(context.Context) (*Span, error)) *MockHeimdallFetchLatestSpanCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// FetchMilestonesFromBlock mocks base method.
func (m *MockHeimdall) FetchMilestonesFromBlock(arg0 context.Context, arg1 uint64) (Waypoints, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchMilestonesFromBlock", arg0, arg1)
	ret0, _ := ret[0].(Waypoints)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchMilestonesFromBlock indicates an expected call of FetchMilestonesFromBlock.
func (mr *MockHeimdallMockRecorder) FetchMilestonesFromBlock(arg0, arg1 any) *MockHeimdallFetchMilestonesFromBlockCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchMilestonesFromBlock", reflect.TypeOf((*MockHeimdall)(nil).FetchMilestonesFromBlock), arg0, arg1)
	return &MockHeimdallFetchMilestonesFromBlockCall{Call: call}
}

// MockHeimdallFetchMilestonesFromBlockCall wrap *gomock.Call
type MockHeimdallFetchMilestonesFromBlockCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHeimdallFetchMilestonesFromBlockCall) Return(arg0 Waypoints, arg1 error) *MockHeimdallFetchMilestonesFromBlockCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHeimdallFetchMilestonesFromBlockCall) Do(f func(context.Context, uint64) (Waypoints, error)) *MockHeimdallFetchMilestonesFromBlockCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHeimdallFetchMilestonesFromBlockCall) DoAndReturn(f func(context.Context, uint64) (Waypoints, error)) *MockHeimdallFetchMilestonesFromBlockCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// OnMilestoneEvent mocks base method.
func (m *MockHeimdall) OnMilestoneEvent(arg0 func(*Milestone)) polygoncommon.UnregisterFunc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OnMilestoneEvent", arg0)
	ret0, _ := ret[0].(polygoncommon.UnregisterFunc)
	return ret0
}

// OnMilestoneEvent indicates an expected call of OnMilestoneEvent.
func (mr *MockHeimdallMockRecorder) OnMilestoneEvent(arg0 any) *MockHeimdallOnMilestoneEventCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnMilestoneEvent", reflect.TypeOf((*MockHeimdall)(nil).OnMilestoneEvent), arg0)
	return &MockHeimdallOnMilestoneEventCall{Call: call}
}

// MockHeimdallOnMilestoneEventCall wrap *gomock.Call
type MockHeimdallOnMilestoneEventCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHeimdallOnMilestoneEventCall) Return(arg0 polygoncommon.UnregisterFunc) *MockHeimdallOnMilestoneEventCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHeimdallOnMilestoneEventCall) Do(f func(func(*Milestone)) polygoncommon.UnregisterFunc) *MockHeimdallOnMilestoneEventCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHeimdallOnMilestoneEventCall) DoAndReturn(f func(func(*Milestone)) polygoncommon.UnregisterFunc) *MockHeimdallOnMilestoneEventCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// OnSpanEvent mocks base method.
func (m *MockHeimdall) OnSpanEvent(arg0 func(*Span)) polygoncommon.UnregisterFunc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OnSpanEvent", arg0)
	ret0, _ := ret[0].(polygoncommon.UnregisterFunc)
	return ret0
}

// OnSpanEvent indicates an expected call of OnSpanEvent.
func (mr *MockHeimdallMockRecorder) OnSpanEvent(arg0 any) *MockHeimdallOnSpanEventCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnSpanEvent", reflect.TypeOf((*MockHeimdall)(nil).OnSpanEvent), arg0)
	return &MockHeimdallOnSpanEventCall{Call: call}
}

// MockHeimdallOnSpanEventCall wrap *gomock.Call
type MockHeimdallOnSpanEventCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHeimdallOnSpanEventCall) Return(arg0 polygoncommon.UnregisterFunc) *MockHeimdallOnSpanEventCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHeimdallOnSpanEventCall) Do(f func(func(*Span)) polygoncommon.UnregisterFunc) *MockHeimdallOnSpanEventCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHeimdallOnSpanEventCall) DoAndReturn(f func(func(*Span)) polygoncommon.UnregisterFunc) *MockHeimdallOnSpanEventCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
