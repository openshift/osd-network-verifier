package kube

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	fak8s "k8s.io/client-go/kubernetes/fake"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/probes/legacy"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

func TestNewKubeVerifier(t *testing.T) {
	clientset := fak8s.NewClientset()

	tests := []struct {
		name    string
		debug   bool
		wantErr bool
	}{
		{
			name:    "create verifier with debug enabled",
			debug:   true,
			wantErr: false,
		},
		{
			name:    "create verifier with debug disabled",
			debug:   false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := NewKubeVerifier(clientset, tt.debug)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKubeVerifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if verifier == nil {
					t.Error("NewKubeVerifier() returned nil verifier")
				} else {
					if verifier.Logger == nil {
						t.Error("NewKubeVerifier() returned verifier with nil logger")
					}
				}
			}
		})
	}
}

func TestKubeVerifier_ValidateEgress(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	tests := []struct {
		name        string
		input       verifier.ValidateEgressInput
		wantSuccess bool
		wantErrors  bool
	}{
		{
			name: "successful validation with curl probe",
			input: verifier.ValidateEgressInput{
				Ctx:            context.Background(),
				PlatformType:   cloud.AWSClassic,
				Probe:          curl.Probe{},
				EgressListYaml: "https://example.com:443\n",
				Timeout:        30 * time.Second,
				AWS:            verifier.AwsEgressConfig{Region: "us-east-1"},
				Proxy:          proxy.ProxyConfig{},
			},
			wantSuccess: false, // Will fail because job won't complete in test
			wantErrors:  true,
		},
		{
			name: "reject non-curl probe",
			input: verifier.ValidateEgressInput{
				Ctx:            context.Background(),
				PlatformType:   cloud.AWSClassic,
				Probe:          legacy.Probe{},
				EgressListYaml: "https://example.com:443\n",
				Timeout:        30 * time.Second,
				AWS:            verifier.AwsEgressConfig{Region: "us-east-1"},
			},
			wantSuccess: false,
			wantErrors:  true,
		},
		{
			name: "use default timeout when zero",
			input: verifier.ValidateEgressInput{
				Ctx:            context.Background(),
				PlatformType:   cloud.AWSClassic,
				Probe:          curl.Probe{},
				EgressListYaml: "https://example.com:443\n",
				Timeout:        0,
				AWS:            verifier.AwsEgressConfig{Region: "us-east-1"},
			},
			wantSuccess: false,
			wantErrors:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kubeVerifier.ValidateEgress(tt.input)

			if result == nil {
				t.Error("ValidateEgress() returned nil output")
				return
			}

			if tt.wantSuccess && !result.IsSuccessful() {
				t.Errorf("ValidateEgress() expected success but got failure")
			}

			if tt.wantErrors {
				failures, exceptions, errors := result.Parse()
				if len(failures) == 0 && len(exceptions) == 0 && len(errors) == 0 {
					t.Errorf("ValidateEgress() expected errors but got none")
				}
			}
		})
	}
}

func TestKubeVerifier_generateCurlCommands(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	tests := []struct {
		name                     string
		egressListStr            string
		tlsDisabledEgressListStr string
		timeout                  time.Duration
		proxyConfig              proxy.ProxyConfig
		wantErr                  bool
	}{
		{
			name:                     "basic curl command generation",
			egressListStr:            "https://example.com:443",
			tlsDisabledEgressListStr: "",
			timeout:                  30 * time.Second,
			proxyConfig:              proxy.ProxyConfig{},
			wantErr:                  false,
		},
		{
			name:                     "with TLS disabled URLs",
			egressListStr:            "https://example.com:443",
			tlsDisabledEgressListStr: "http://insecure.example.com:80",
			timeout:                  60 * time.Second,
			proxyConfig:              proxy.ProxyConfig{NoTls: true},
			wantErr:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := kubeVerifier.generateCurlCommands(
				tt.egressListStr,
				tt.tlsDisabledEgressListStr,
				tt.timeout,
				tt.proxyConfig,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateCurlCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cmd == "" {
				t.Error("generateCurlCommands() returned empty command")
			}
		})
	}
}

func TestKubeVerifier_buildJobSpec(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	input := createJobInput{
		JobName:                 "test-job",
		Namespace:               "test-namespace",
		ContainerImage:          "test-image:latest",
		CurlCommand:             "curl https://example.com",
		ProxySettings:           map[string]string{"HTTP_PROXY": "http://proxy:8080"},
		TTLSecondsAfterFinished: 600,
		ActiveDeadlineSeconds:   300,
		BackoffLimit:            0,
		ResourceLimits:          corev1.ResourceRequirements{},
		Ctx:                     context.Background(),
	}

	job := kubeVerifier.buildJobSpec(input)

	if job == nil {
		t.Fatal("buildJobSpec() returned nil job")
	}

	if job.Name != input.JobName {
		t.Errorf("buildJobSpec() job name = %v, want %v", job.Name, input.JobName)
	}

	if job.Namespace != input.Namespace {
		t.Errorf("buildJobSpec() job namespace = %v, want %v", job.Namespace, input.Namespace)
	}

	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("buildJobSpec() expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
	}

	container := job.Spec.Template.Spec.Containers[0]
	if container.Name != "osd-network-verifier" {
		t.Errorf("buildJobSpec() container name = %v, want %v", container.Name, "osd-network-verifier")
	}

	if container.Image != input.ContainerImage {
		t.Errorf("buildJobSpec() container image = %v, want %v", container.Image, input.ContainerImage)
	}

	if len(container.Args) != 1 || container.Args[0] != input.CurlCommand {
		t.Errorf("buildJobSpec() container args = %v, want [%v]", container.Args, input.CurlCommand)
	}

	// Check security context
	if container.SecurityContext == nil {
		t.Error("buildJobSpec() container security context is nil")
	} else {
		if container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
			t.Error("buildJobSpec() container should not allow privilege escalation")
		}
		if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
			t.Error("buildJobSpec() container should have read-only root filesystem")
		}
		if container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
			t.Error("buildJobSpec() container should run as non-root")
		}
	}

	// Check TTL
	if job.Spec.TTLSecondsAfterFinished == nil || *job.Spec.TTLSecondsAfterFinished != input.TTLSecondsAfterFinished {
		t.Errorf("buildJobSpec() TTL = %v, want %v", job.Spec.TTLSecondsAfterFinished, input.TTLSecondsAfterFinished)
	}
}

func TestKubeVerifier_buildEnvVars(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	tests := []struct {
		name          string
		proxySettings map[string]string
		wantCount     int
	}{
		{
			name:          "no proxy settings",
			proxySettings: map[string]string{},
			wantCount:     0,
		},
		{
			name: "with proxy settings",
			proxySettings: map[string]string{
				"HTTP_PROXY":  "http://proxy:8080",
				"HTTPS_PROXY": "https://proxy:8080",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
			wantCount: 3,
		},
		{
			name: "with empty values",
			proxySettings: map[string]string{
				"HTTP_PROXY":  "http://proxy:8080",
				"HTTPS_PROXY": "",
				"NO_PROXY":    "localhost",
			},
			wantCount: 2, // Empty values should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := kubeVerifier.buildEnvVars(tt.proxySettings)

			if len(envVars) != tt.wantCount {
				t.Errorf("buildEnvVars() returned %d env vars, want %d", len(envVars), tt.wantCount)
			}

			// Verify that non-empty values are included
			for key, value := range tt.proxySettings {
				if value != "" {
					found := false
					for _, env := range envVars {
						if env.Name == key && env.Value == value {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("buildEnvVars() missing env var %s=%s", key, value)
					}
				}
			}
		})
	}
}

func TestKubeVerifier_parseJobLogs(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	tests := []struct {
		name     string
		logs     string
		expected string
	}{
		{
			name: "valid logs with separators",
			logs: `Starting job
@NV@
{"url": "https://example.com", "success": true}
@NV@
Job completed`,
			expected: `@NV@
@NV@`,
		},
		{
			name: "logs without separators",
			logs: `Starting job
Some output
Job completed`,
			expected: "",
		},
		{
			name:     "empty logs",
			logs:     "",
			expected: "",
		},
		{
			name: "real world sample with many @NV@ lines",
			logs: `Starting job
@NV@line1
@NV@line2
@NV@line3
Some other log output
@NV@line4
@NV@line5
Job completed`,
			expected: `@NV@line1
@NV@line2
@NV@line3
@NV@line4
@NV@line5`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kubeVerifier.parseJobLogs(tt.logs)
			if result != tt.expected {
				t.Errorf("parseJobLogs() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestKubeVerifier_buildProxyEnvironment(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	tests := []struct {
		name        string
		proxyConfig proxy.ProxyConfig
		expected    map[string]string
	}{
		{
			name:        "empty proxy config",
			proxyConfig: proxy.ProxyConfig{},
			expected:    map[string]string{},
		},
		{
			name: "HTTP proxy only",
			proxyConfig: proxy.ProxyConfig{
				HttpProxy: "http://proxy:8080",
			},
			expected: map[string]string{
				"HTTP_PROXY": "http://proxy:8080",
				"http_proxy": "http://proxy:8080",
			},
		},
		{
			name: "HTTPS proxy only",
			proxyConfig: proxy.ProxyConfig{
				HttpsProxy: "https://proxy:8080",
			},
			expected: map[string]string{
				"HTTPS_PROXY": "https://proxy:8080",
				"https_proxy": "https://proxy:8080",
			},
		},
		{
			name: "full proxy config",
			proxyConfig: proxy.ProxyConfig{
				HttpProxy:  "http://proxy:8080",
				HttpsProxy: "https://proxy:8080",
				NoProxy:    []string{"localhost", "127.0.0.1", ".local"},
			},
			expected: map[string]string{
				"HTTP_PROXY":  "http://proxy:8080",
				"http_proxy":  "http://proxy:8080",
				"HTTPS_PROXY": "https://proxy:8080",
				"https_proxy": "https://proxy:8080",
				"NO_PROXY":    "localhost,127.0.0.1,.local",
				"no_proxy":    "localhost,127.0.0.1,.local",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kubeVerifier.buildProxyEnvironment(tt.proxyConfig)

			if len(result) != len(tt.expected) {
				t.Errorf("buildProxyEnvironment() returned %d items, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("buildProxyEnvironment() missing key %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("buildProxyEnvironment() key %s = %s, want %s", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestKubeVerifier_buildResourceRequirements(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	resources := kubeVerifier.buildResourceRequirements()

	// Check CPU limits
	cpuLimit := resources.Limits[corev1.ResourceCPU]
	expectedCPULimit := resource.MustParse("100m")
	if cpuLimit.Cmp(expectedCPULimit) != 0 {
		t.Errorf("buildResourceRequirements() CPU limit = %v, want %v", cpuLimit, expectedCPULimit)
	}

	// Check memory limits
	memoryLimit := resources.Limits[corev1.ResourceMemory]
	expectedMemoryLimit := resource.MustParse("128Mi")
	if memoryLimit.Cmp(expectedMemoryLimit) != 0 {
		t.Errorf("buildResourceRequirements() memory limit = %v, want %v", memoryLimit, expectedMemoryLimit)
	}

	// Check CPU requests
	cpuRequest := resources.Requests[corev1.ResourceCPU]
	expectedCPURequest := resource.MustParse("50m")
	if cpuRequest.Cmp(expectedCPURequest) != 0 {
		t.Errorf("buildResourceRequirements() CPU request = %v, want %v", cpuRequest, expectedCPURequest)
	}

	// Check memory requests
	memoryRequest := resources.Requests[corev1.ResourceMemory]
	expectedMemoryRequest := resource.MustParse("64Mi")
	if memoryRequest.Cmp(expectedMemoryRequest) != 0 {
		t.Errorf("buildResourceRequirements() memory request = %v, want %v", memoryRequest, expectedMemoryRequest)
	}
}

func TestKubeVerifier_collectAndParseJobOutput(t *testing.T) {
	mockKubeClient := NewMockClient()
	kubeVerifier := &KubeVerifier{
		KubeClient: mockKubeClient,
		Logger:     &ocmlog.GlogLogger{},
		Output:     output.Output{},
	}

	tests := []struct {
		name        string
		jobName     string
		logs        string
		logError    error
		probe       curl.Probe
		wantErr     bool
		wantSuccess bool
	}{
		{
			name:    "successful log collection and parsing",
			jobName: "test-job",
			logs: `@NV@
{"url": "https://example.com", "exit_code": 0}
@NV@`,
			logError:    nil,
			probe:       curl.Probe{},
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name:        "error getting logs",
			jobName:     "test-job",
			logs:        "",
			logError:    errors.New("failed to get logs"),
			probe:       curl.Probe{},
			wantErr:     true,
			wantSuccess: false,
		},
		{
			name:    "no valid probe output",
			jobName: "test-job",
			logs: `Starting job
Some output without separators
Job completed`,
			logError:    nil,
			probe:       curl.Probe{},
			wantErr:     true,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubeClient.SetGetJobLogsResult(tt.logs, tt.logError)

			err := kubeVerifier.collectAndParseJobOutput(context.Background(), tt.jobName, tt.probe)

			if (err != nil) != tt.wantErr {
				t.Errorf("collectAndParseJobOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKubeVerifier_createAndExecuteJob_MockedClient(t *testing.T) {
	mockKubeClient := NewMockClient()
	kubeVerifier := &KubeVerifier{
		KubeClient: mockKubeClient,
		Logger:     &ocmlog.GlogLogger{},
		Output:     output.Output{},
	}

	jobInput := createJobInput{
		JobName:   "test-job",
		Namespace: "test-namespace",
		Ctx:       context.Background(),
	}

	tests := []struct {
		name        string
		createError error
		waitError   error
		wantErr     bool
	}{
		{
			name:        "successful job creation and execution",
			createError: nil,
			waitError:   nil,
			wantErr:     false,
		},
		{
			name:        "error creating job",
			createError: errors.New("failed to create job"),
			waitError:   nil,
			wantErr:     true,
		},
		{
			name:        "error waiting for job completion",
			createError: nil,
			waitError:   errors.New("job failed or timed out"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset output and mock state for each test
			kubeVerifier.Output = output.Output{}
			mockKubeClient.SetCreateJobError(tt.createError)
			mockKubeClient.SetWaitForJobCompletionError(tt.waitError)

			err := kubeVerifier.createAndExecuteJob(jobInput)

			if (err != nil) != tt.wantErr {
				t.Errorf("createAndExecuteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKubeVerifier_writeDebugLogs(t *testing.T) {
	clientset := fak8s.NewClientset()
	kubeVerifier, err := NewKubeVerifier(clientset, false)
	if err != nil {
		t.Fatalf("Failed to create KubeVerifier: %v", err)
	}

	testLog := "test debug message"

	// This should not panic or error
	kubeVerifier.writeDebugLogs(testLog)

	// Check that the log was added to the output
	// Since output doesn't expose debug logs directly, we check via the Format method
	output := kubeVerifier.Output.Format(true)
	if !strings.Contains(output, testLog) {
		t.Errorf("writeDebugLogs() did not add log to output debug logs")
	}
}
