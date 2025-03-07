// Code generated by MockGen. DO NOT EDIT.
// Source: model_instance.go
//
// Generated by this command:
//
//	mockgen -source model_instance.go -destination model_instance_mocks.go -package runner
//

// Package runner is a generated GoMock package.
package runner

import (
	context "context"
	reflect "reflect"

	model "github.com/helixml/helix/api/pkg/model"
	types "github.com/helixml/helix/api/pkg/types"
	gomock "go.uber.org/mock/gomock"
)

// MockModelInstance is a mock of ModelInstance interface.
type MockModelInstance struct {
	ctrl     *gomock.Controller
	recorder *MockModelInstanceMockRecorder
	isgomock struct{}
}

// MockModelInstanceMockRecorder is the mock recorder for MockModelInstance.
type MockModelInstanceMockRecorder struct {
	mock *MockModelInstance
}

// NewMockModelInstance creates a new mock instance.
func NewMockModelInstance(ctrl *gomock.Controller) *MockModelInstance {
	mock := &MockModelInstance{ctrl: ctrl}
	mock.recorder = &MockModelInstanceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelInstance) EXPECT() *MockModelInstanceMockRecorder {
	return m.recorder
}

// Done mocks base method.
func (m *MockModelInstance) Done() <-chan bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Done")
	ret0, _ := ret[0].(<-chan bool)
	return ret0
}

// Done indicates an expected call of Done.
func (mr *MockModelInstanceMockRecorder) Done() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Done", reflect.TypeOf((*MockModelInstance)(nil).Done))
}

// Filter mocks base method.
func (m *MockModelInstance) Filter() types.SessionFilter {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Filter")
	ret0, _ := ret[0].(types.SessionFilter)
	return ret0
}

// Filter indicates an expected call of Filter.
func (mr *MockModelInstanceMockRecorder) Filter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Filter", reflect.TypeOf((*MockModelInstance)(nil).Filter))
}

// GetState mocks base method.
func (m *MockModelInstance) GetState() (*types.ModelInstanceState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetState")
	ret0, _ := ret[0].(*types.ModelInstanceState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetState indicates an expected call of GetState.
func (mr *MockModelInstanceMockRecorder) GetState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetState", reflect.TypeOf((*MockModelInstance)(nil).GetState))
}

// ID mocks base method.
func (m *MockModelInstance) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockModelInstanceMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockModelInstance)(nil).ID))
}

// IsActive mocks base method.
func (m *MockModelInstance) IsActive() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsActive")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsActive indicates an expected call of IsActive.
func (mr *MockModelInstanceMockRecorder) IsActive() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsActive", reflect.TypeOf((*MockModelInstance)(nil).IsActive))
}

// Model mocks base method.
func (m *MockModelInstance) Model() model.Model {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Model")
	ret0, _ := ret[0].(model.Model)
	return ret0
}

// Model indicates an expected call of Model.
func (mr *MockModelInstanceMockRecorder) Model() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Model", reflect.TypeOf((*MockModelInstance)(nil).Model))
}

// QueueSession mocks base method.
func (m *MockModelInstance) QueueSession(session *types.Session, isInitialSession bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "QueueSession", session, isInitialSession)
}

// QueueSession indicates an expected call of QueueSession.
func (mr *MockModelInstanceMockRecorder) QueueSession(session, isInitialSession any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueSession", reflect.TypeOf((*MockModelInstance)(nil).QueueSession), session, isInitialSession)
}

// Stale mocks base method.
func (m *MockModelInstance) Stale() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stale")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Stale indicates an expected call of Stale.
func (mr *MockModelInstanceMockRecorder) Stale() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stale", reflect.TypeOf((*MockModelInstance)(nil).Stale))
}

// Start mocks base method.
func (m *MockModelInstance) Start(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockModelInstanceMockRecorder) Start(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockModelInstance)(nil).Start), ctx)
}

// Stop mocks base method.
func (m *MockModelInstance) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockModelInstanceMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockModelInstance)(nil).Stop))
}
