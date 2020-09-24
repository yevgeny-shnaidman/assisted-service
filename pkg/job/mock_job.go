// Code generated by MockGen. DO NOT EDIT.
// Source: job.go

// Package job is a generated GoMock package.
package job

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	common "github.com/openshift/assisted-service/internal/common"
	events "github.com/openshift/assisted-service/internal/events"
	runtime "k8s.io/apimachinery/pkg/runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockAPI is a mock of API interface
type MockAPI struct {
	ctrl     *gomock.Controller
	recorder *MockAPIMockRecorder
}

// MockAPIMockRecorder is the mock recorder for MockAPI
type MockAPIMockRecorder struct {
	mock *MockAPI
}

// NewMockAPI creates a new mock instance
func NewMockAPI(ctrl *gomock.Controller) *MockAPI {
	mock := &MockAPI{ctrl: ctrl}
	mock.recorder = &MockAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAPI) EXPECT() *MockAPIMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockAPI) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, obj}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Create", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockAPIMockRecorder) Create(ctx, obj interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, obj}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAPI)(nil).Create), varargs...)
}

// Monitor mocks base method
func (m *MockAPI) Monitor(ctx context.Context, name, namespace string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Monitor", ctx, name, namespace)
	ret0, _ := ret[0].(error)
	return ret0
}

// Monitor indicates an expected call of Monitor
func (mr *MockAPIMockRecorder) Monitor(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Monitor", reflect.TypeOf((*MockAPI)(nil).Monitor), ctx, name, namespace)
}

// Delete mocks base method
func (m *MockAPI) Delete(ctx context.Context, name, namespace string, force bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, name, namespace, force)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockAPIMockRecorder) Delete(ctx, name, namespace, force interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockAPI)(nil).Delete), ctx, name, namespace, force)
}

// GenerateISO mocks base method
func (m *MockAPI) GenerateISO(ctx context.Context, cluster common.Cluster, jobName, imageName, ignitionConfig string, eventsHandler events.Handler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateISO", ctx, cluster, jobName, imageName, ignitionConfig, eventsHandler)
	ret0, _ := ret[0].(error)
	return ret0
}

// GenerateISO indicates an expected call of GenerateISO
func (mr *MockAPIMockRecorder) GenerateISO(ctx, cluster, jobName, imageName, ignitionConfig, eventsHandler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateISO", reflect.TypeOf((*MockAPI)(nil).GenerateISO), ctx, cluster, jobName, imageName, ignitionConfig, eventsHandler)
}

// GenerateInstallConfig mocks base method
func (m *MockAPI) GenerateInstallConfig(ctx context.Context, cluster common.Cluster, cfg []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateInstallConfig", ctx, cluster, cfg)
	ret0, _ := ret[0].(error)
	return ret0
}

// GenerateInstallConfig indicates an expected call of GenerateInstallConfig
func (mr *MockAPIMockRecorder) GenerateInstallConfig(ctx, cluster, cfg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateInstallConfig", reflect.TypeOf((*MockAPI)(nil).GenerateInstallConfig), ctx, cluster, cfg)
}

// AbortInstallConfig mocks base method
func (m *MockAPI) AbortInstallConfig(ctx context.Context, cluster common.Cluster) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AbortInstallConfig", ctx, cluster)
	ret0, _ := ret[0].(error)
	return ret0
}

// AbortInstallConfig indicates an expected call of AbortInstallConfig
func (mr *MockAPIMockRecorder) AbortInstallConfig(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AbortInstallConfig", reflect.TypeOf((*MockAPI)(nil).AbortInstallConfig), ctx, cluster)
}
