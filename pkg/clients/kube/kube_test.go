package kube

import (
	"context"
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fak8s "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const testNamespace = "test-ns"

func TestClient_CreateJob(t *testing.T) {
	type args struct {
		job *batchv1.Job
	}
	tests := []struct {
		name           string
		args           args
		want           *batchv1.Job
		wantErr        bool
		checkJobExists bool
		checkPodExists bool
	}{
		{
			name: "create a job",
			args: args{
				job: getMockedJob("my-job"),
			},
			want:           getMockedJob("my-job"),
			checkJobExists: true,
			checkPodExists: true,
		},
		{
			name: "error when creating an empty job",
			args: args{
				job: &batchv1.Job{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		// Create a empty mock client for each test
		mockedClientset := fak8s.NewClientset()
		setupJobMockingReactors(mockedClientset)

		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				clientset: mockedClientset,
				namespace: testNamespace,
			}
			got, err := c.CreateJob(context.Background(), tt.args.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.CreateJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.CreateJob() = %v, want %v", got, tt.want)
			}
			if tt.checkJobExists {
				_, err := mockedClientset.BatchV1().Jobs(c.namespace).Get(context.Background(), tt.args.job.Name, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Could not confirm job was created: %v", err)
				}

			}
			if tt.checkPodExists {
				_, err := mockedClientset.CoreV1().Pods(c.namespace).Get(context.Background(), tt.args.job.Name, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Could not confirm pod was created: %v", err)
				}

			}
		})
	}
}

func TestClient_GetJob(t *testing.T) {
	// Create a mock client with a mock job and pod
	mockedClientset := fak8s.NewClientset()
	setupJobMockingReactors(mockedClientset)
	createMockedJob("osd-network-verifier", mockedClientset)

	type args struct {
		jobName string
	}
	tests := []struct {
		name    string
		args    args
		want    *batchv1.Job
		wantErr bool
	}{
		{
			name: "get the mocked job",
			args: args{
				jobName: "osd-network-verifier",
			},
			want: getMockedJob("osd-network-verifier"),
		},
		{
			name: "error when getting invalid job",
			args: args{
				jobName: "invalid-job",
			},
			want:    &batchv1.Job{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				clientset: mockedClientset,
				namespace: testNamespace,
			}
			got, err := c.GetJob(context.Background(), tt.args.jobName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetJobLogs(t *testing.T) {
	// Create a mock client with a mock job and pod
	mockedClientset := fak8s.NewClientset()
	setupJobMockingReactors(mockedClientset)
	createMockedJob("osd-network-verifier", mockedClientset)

	type args struct {
		jobName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "fetch the mock logs of a valid mock job",
			args: args{
				jobName: "osd-network-verifier",
			},
			want:    "fake logs",
			wantErr: false,
		},
		{
			name: "attempt to fetch the logs of an non-existent mock job",
			args: args{
				jobName: "invalid-job",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				clientset: mockedClientset,
				namespace: testNamespace,
			}
			got, err := c.GetJobLogs(context.Background(), tt.args.jobName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetJobLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.GetJobLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_DeleteJob(t *testing.T) {
	// Create a mock client with a mock job and pod
	mockedClientset := fak8s.NewClientset()
	setupJobMockingReactors(mockedClientset)
	createMockedJob("osd-network-verifier", mockedClientset)

	type args struct {
		jobName string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		checkPodDeleted bool
	}{
		{
			name: "delete the mocked job",
			args: args{
				jobName: "osd-network-verifier",
			},
			checkPodDeleted: true,
		},
		{
			name: "error when deleting invalid job",
			args: args{
				jobName: "invalid-job",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				clientset: mockedClientset,
				namespace: testNamespace,
			}

			if tt.checkPodDeleted {
				// Check if pod exists before deletion
				_, err := mockedClientset.CoreV1().Pods(testNamespace).Get(context.Background(), tt.args.jobName, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					t.Errorf("Pod doesn't exist before job deletion")
				}
			}

			// Delete job
			if err := c.DeleteJob(context.Background(), tt.args.jobName); (err != nil) != tt.wantErr {
				t.Errorf("Client.DeleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkPodDeleted {
				// Check if pod is deleted after job deletion
				_, err := mockedClientset.CoreV1().Pods(testNamespace).Get(context.Background(), tt.args.jobName, metav1.GetOptions{})
				if err == nil || !errors.IsNotFound(err) {
					t.Errorf("Pod seems to still exist after job deletion or error while checking for pod: %v", err)
				}
			}
		})
	}
}

// getMockedJob returns a K8s Job spec with the given name in the test namespace
func getMockedJob(name string) *batchv1.Job {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
	}
}

// setupJobMockingReactors adds hooks ("reactors") to the fake clientset that emulates the behavior of the KubeAPI
// (i.e., when a job is created, a pod is created and when a job is deleted, the pod is deleted)
func setupJobMockingReactors(clientset *fak8s.Clientset) {
	// Add hook to job creation that creates an associated pod
	clientset.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Create the associated pod when the job is created
		createdJob := action.(k8stesting.CreateAction).GetObject().(*batchv1.Job)
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: createdJob.Namespace,
				Name:      createdJob.Name,
				Labels: map[string]string{
					"job-name": createdJob.Name,
				},
			},
		}
		err := clientset.Tracker().Create(corev1.SchemeGroupVersion.WithResource("pods"), pod, createdJob.Namespace)
		return false, nil, err
	})

	// Add hook to job deletion that also deletes the associated pod if propagation policy is set properly
	clientset.PrependReactor("delete", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Gather delete action details
		deleteAction := action.(k8stesting.DeleteAction)
		deletedJobNamespace := deleteAction.GetNamespace()
		deletedJobName := deleteAction.GetName()
		deletedJobPropagationPolicy := deleteAction.GetDeleteOptions().PropagationPolicy

		// Delete the associated pod if the propagation policy is set properly
		if deletedJobPropagationPolicy != nil && *deletedJobPropagationPolicy != metav1.DeletePropagationOrphan {
			clientset.Tracker().Delete(corev1.SchemeGroupVersion.WithResource("pods"), deletedJobNamespace, deletedJobName)
		}
		return false, nil, nil
	})
}

// createMockedJob creates a mock job in the test namespace
func createMockedJob(name string, clientset *fak8s.Clientset) {
	clientset.BatchV1().Jobs(testNamespace).Create(context.Background(), getMockedJob(name), metav1.CreateOptions{})
}
