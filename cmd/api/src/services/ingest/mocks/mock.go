// Copyright 2023 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/specterops/bloodhound/src/services/ingest (interfaces: IngestData)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/specterops/bloodhound/src/model"
	gomock "go.uber.org/mock/gomock"
)

// MockIngestData is a mock of IngestData interface.
type MockIngestData struct {
	ctrl     *gomock.Controller
	recorder *MockIngestDataMockRecorder
}

// MockIngestDataMockRecorder is the mock recorder for MockIngestData.
type MockIngestDataMockRecorder struct {
	mock *MockIngestData
}

// NewMockIngestData creates a new mock instance.
func NewMockIngestData(ctrl *gomock.Controller) *MockIngestData {
	mock := &MockIngestData{ctrl: ctrl}
	mock.recorder = &MockIngestDataMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIngestData) EXPECT() *MockIngestDataMockRecorder {
	return m.recorder
}

// CancelAllIngestJobs mocks base method.
func (m *MockIngestData) CancelAllIngestJobs(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelAllIngestJobs", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CancelAllIngestJobs indicates an expected call of CancelAllIngestJobs.
func (mr *MockIngestDataMockRecorder) CancelAllIngestJobs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelAllIngestJobs", reflect.TypeOf((*MockIngestData)(nil).CancelAllIngestJobs), arg0)
}

// CreateCompositionInfo mocks base method.
func (m *MockIngestData) CreateCompositionInfo(arg0 context.Context, arg1 model.EdgeCompositionNodes, arg2 model.EdgeCompositionEdges) (model.EdgeCompositionNodes, model.EdgeCompositionEdges, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCompositionInfo", arg0, arg1, arg2)
	ret0, _ := ret[0].(model.EdgeCompositionNodes)
	ret1, _ := ret[1].(model.EdgeCompositionEdges)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateCompositionInfo indicates an expected call of CreateCompositionInfo.
func (mr *MockIngestDataMockRecorder) CreateCompositionInfo(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCompositionInfo", reflect.TypeOf((*MockIngestData)(nil).CreateCompositionInfo), arg0, arg1, arg2)
}

// CreateIngestJob mocks base method.
func (m *MockIngestData) CreateIngestJob(arg0 context.Context, arg1 model.IngestJob) (model.IngestJob, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIngestJob", arg0, arg1)
	ret0, _ := ret[0].(model.IngestJob)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIngestJob indicates an expected call of CreateIngestJob.
func (mr *MockIngestDataMockRecorder) CreateIngestJob(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIngestJob", reflect.TypeOf((*MockIngestData)(nil).CreateIngestJob), arg0, arg1)
}

// CreateIngestTask mocks base method.
func (m *MockIngestData) CreateIngestTask(arg0 context.Context, arg1 model.IngestTask) (model.IngestTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIngestTask", arg0, arg1)
	ret0, _ := ret[0].(model.IngestTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIngestTask indicates an expected call of CreateIngestTask.
func (mr *MockIngestDataMockRecorder) CreateIngestTask(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIngestTask", reflect.TypeOf((*MockIngestData)(nil).CreateIngestTask), arg0, arg1)
}

// DeleteAllIngestJobs mocks base method.
func (m *MockIngestData) DeleteAllIngestJobs(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllIngestJobs", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllIngestJobs indicates an expected call of DeleteAllIngestJobs.
func (mr *MockIngestDataMockRecorder) DeleteAllIngestJobs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllIngestJobs", reflect.TypeOf((*MockIngestData)(nil).DeleteAllIngestJobs), arg0)
}

// DeleteAllIngestTasks mocks base method.
func (m *MockIngestData) DeleteAllIngestTasks(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllIngestTasks", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllIngestTasks indicates an expected call of DeleteAllIngestTasks.
func (mr *MockIngestDataMockRecorder) DeleteAllIngestTasks(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllIngestTasks", reflect.TypeOf((*MockIngestData)(nil).DeleteAllIngestTasks), arg0)
}

// GetAllIngestJobs mocks base method.
func (m *MockIngestData) GetAllIngestJobs(arg0 context.Context, arg1, arg2 int, arg3 string, arg4 model.SQLFilter) ([]model.IngestJob, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllIngestJobs", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].([]model.IngestJob)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetAllIngestJobs indicates an expected call of GetAllIngestJobs.
func (mr *MockIngestDataMockRecorder) GetAllIngestJobs(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllIngestJobs", reflect.TypeOf((*MockIngestData)(nil).GetAllIngestJobs), arg0, arg1, arg2, arg3, arg4)
}

// GetIngestJob mocks base method.
func (m *MockIngestData) GetIngestJob(arg0 context.Context, arg1 int64) (model.IngestJob, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngestJob", arg0, arg1)
	ret0, _ := ret[0].(model.IngestJob)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngestJob indicates an expected call of GetIngestJob.
func (mr *MockIngestDataMockRecorder) GetIngestJob(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngestJob", reflect.TypeOf((*MockIngestData)(nil).GetIngestJob), arg0, arg1)
}

// GetIngestJobsWithStatus mocks base method.
func (m *MockIngestData) GetIngestJobsWithStatus(arg0 context.Context, arg1 model.JobStatus) ([]model.IngestJob, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngestJobsWithStatus", arg0, arg1)
	ret0, _ := ret[0].([]model.IngestJob)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngestJobsWithStatus indicates an expected call of GetIngestJobsWithStatus.
func (mr *MockIngestDataMockRecorder) GetIngestJobsWithStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngestJobsWithStatus", reflect.TypeOf((*MockIngestData)(nil).GetIngestJobsWithStatus), arg0, arg1)
}

// UpdateIngestJob mocks base method.
func (m *MockIngestData) UpdateIngestJob(arg0 context.Context, arg1 model.IngestJob) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateIngestJob", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateIngestJob indicates an expected call of UpdateIngestJob.
func (mr *MockIngestDataMockRecorder) UpdateIngestJob(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateIngestJob", reflect.TypeOf((*MockIngestData)(nil).UpdateIngestJob), arg0, arg1)
}
