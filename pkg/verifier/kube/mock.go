package kube

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
)

// MockClient is a simple mock implementation for testing
type MockClient struct {
	namespace                 string
	createJobError            error
	waitForJobCompletionError error
	getJobLogsResult          string
	getJobLogsError           error
}

// NewMockClient creates a new simple mock client
func NewMockClient() *MockClient {
	return &MockClient{
		namespace: "test-namespace",
	}
}

// GetNamespace returns the configured namespace
func (m *MockClient) GetNamespace() string {
	return m.namespace
}

// CreateJob mocks job creation
func (m *MockClient) CreateJob(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	if m.createJobError != nil {
		return nil, m.createJobError
	}
	return job, nil
}

func (m *MockClient) DeleteJob(ctx context.Context, jobName string) error {
	return nil
}

// WaitForJobCompletion mocks waiting for job completion
func (m *MockClient) WaitForJobCompletion(ctx context.Context, jobName string) error {
	return m.waitForJobCompletionError
}

// GetJobLogs mocks getting job logs
func (m *MockClient) GetJobLogs(ctx context.Context, jobName string) (string, error) {
	if m.getJobLogsError != nil {
		return "", m.getJobLogsError
	}
	return m.getJobLogsResult, nil
}

// SetNamespace allows setting the namespace for testing
func (m *MockClient) SetNamespace(namespace string) {
	m.namespace = namespace
}

// SetCreateJobError sets the error to return from CreateJob
func (m *MockClient) SetCreateJobError(err error) {
	m.createJobError = err
}

// SetWaitForJobCompletionError sets the error to return from WaitForJobCompletion
func (m *MockClient) SetWaitForJobCompletionError(err error) {
	m.waitForJobCompletionError = err
}

// SetGetJobLogsResult sets the result to return from GetJobLogs
func (m *MockClient) SetGetJobLogsResult(logs string, err error) {
	m.getJobLogsResult = logs
	m.getJobLogsError = err
}
func (m *MockClient) CleanupJob(ctx context.Context, jobName string) error {
	return nil
}
