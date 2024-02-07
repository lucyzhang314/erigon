// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ledgerwatch/erigon/polygon/p2p (interfaces: PeerManager)

// Package p2p is a generated GoMock package.
package p2p

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/ledgerwatch/erigon/core/types"
)

// MockPeerManager is a mock of PeerManager interface.
type MockPeerManager struct {
	ctrl     *gomock.Controller
	recorder *MockPeerManagerMockRecorder
}

// MockPeerManagerMockRecorder is the mock recorder for MockPeerManager.
type MockPeerManagerMockRecorder struct {
	mock *MockPeerManager
}

// NewMockPeerManager creates a new mock instance.
func NewMockPeerManager(ctrl *gomock.Controller) *MockPeerManager {
	mock := &MockPeerManager{ctrl: ctrl}
	mock.recorder = &MockPeerManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPeerManager) EXPECT() *MockPeerManagerMockRecorder {
	return m.recorder
}

// DownloadHeaders mocks base method.
func (m *MockPeerManager) DownloadHeaders(arg0 context.Context, arg1, arg2 uint64, arg3 PeerId) ([]*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DownloadHeaders", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DownloadHeaders indicates an expected call of DownloadHeaders.
func (mr *MockPeerManagerMockRecorder) DownloadHeaders(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DownloadHeaders", reflect.TypeOf((*MockPeerManager)(nil).DownloadHeaders), arg0, arg1, arg2, arg3)
}

// MaxPeers mocks base method.
func (m *MockPeerManager) MaxPeers() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaxPeers")
	ret0, _ := ret[0].(int)
	return ret0
}

// MaxPeers indicates an expected call of MaxPeers.
func (mr *MockPeerManagerMockRecorder) MaxPeers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaxPeers", reflect.TypeOf((*MockPeerManager)(nil).MaxPeers))
}

// PeerBlockNumInfos mocks base method.
func (m *MockPeerManager) PeerBlockNumInfos() PeerBlockNumInfos {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PeerBlockNumInfos")
	ret0, _ := ret[0].(PeerBlockNumInfos)
	return ret0
}

// PeerBlockNumInfos indicates an expected call of PeerBlockNumInfos.
func (mr *MockPeerManagerMockRecorder) PeerBlockNumInfos() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PeerBlockNumInfos", reflect.TypeOf((*MockPeerManager)(nil).PeerBlockNumInfos))
}

// Penalize mocks base method.
func (m *MockPeerManager) Penalize(arg0 PeerId) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Penalize", arg0)
}

// Penalize indicates an expected call of Penalize.
func (mr *MockPeerManagerMockRecorder) Penalize(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Penalize", reflect.TypeOf((*MockPeerManager)(nil).Penalize), arg0)
}
