package kube

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/kube"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/curlgen"
	"github.com/openshift/osd-network-verifier/pkg/data/egress_lists"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

const (
	defaultContainerImage          = "image-registry.openshift-image-registry.svc:5000/openshift/tools"
	defaultTTLSecondsAfterFinished = int32(600) // 10 minutes
	defaultActiveDeadlineSeconds   = int32(300) // 5 minutes
	defaultBackoffLimit            = int32(0)   // We only want to try once
	defaultCurlTimeout             = 30 * time.Second
)

type KubeVerifier struct {
	KubeClient kube.Client
	Logger     ocmlog.Logger
	Output     output.Output
}

type createJobInput struct {
	JobName                 string
	Namespace               string
	ContainerImage          string
	CurlCommand             string
	ProxySettings           map[string]string
	TTLSecondsAfterFinished int32
	ActiveDeadlineSeconds   int32
	BackoffLimit            int32
	ResourceLimits          corev1.ResourceRequirements
	Ctx                     context.Context
}

// NewKubeVerifier returns a KubeVerifier authenticated to a k8s cluster with the given clientset
func NewKubeVerifier(clientset *kubernetes.Clientset, debug bool) (*KubeVerifier, error) {
	builder := ocmlog.NewStdLoggerBuilder()
	builder.Debug(debug)
	logger, err := builder.Build()
	if err != nil {
		return &KubeVerifier{}, fmt.Errorf("unable to build logger: %s", err.Error())
	}

	kubeClient, err := kube.NewClient(clientset)
	if err != nil {
		return &KubeVerifier{}, err
	}

	return &KubeVerifier{
		KubeClient: *kubeClient,
		Logger:     logger,
		Output:     output.Output{},
	}, nil
}

func (k *KubeVerifier) ValidateEgress(vei verifier.ValidateEgressInput) *output.Output {
	// Validate cloud platform type
	if !vei.PlatformType.IsValid() {
		vei.PlatformType = cloud.AWSClassic
	}

	if _, ok := vei.Probe.(curl.Probe); !ok {
		return k.Output.AddError(errors.New("verification via pod mode only supports curl probe"))
	}

	if vei.Timeout <= 0 {
		vei.Timeout = defaultCurlTimeout
	}
	k.writeDebugLogs(vei.Ctx, fmt.Sprintf("configured a %s timeout for each egress request", vei.Timeout))

	// Generate egress lists for the given PlatformType
	generatorVariables := map[string]string{"AWS_REGION": vei.AWS.Region}
	generator := egress_lists.NewGenerator(vei.PlatformType, generatorVariables, k.Logger)
	egressListStr, tlsDisabledEgressListStr, err := generator.GenerateEgressLists(vei.Ctx, vei.EgressListYaml)
	if err != nil {
		return k.Output.AddError(err)
	}

	// Generate curl commands
	curlCommand, err := k.generateCurlCommands(egressListStr, tlsDisabledEgressListStr, vei.Timeout, vei.Proxy)
	if err != nil {
		return k.Output.AddError(err)
	}

	// Create and execute Job
	jobName := fmt.Sprintf("osd-network-verifier-job-%d", time.Now().Unix())
	proxySettings := k.buildProxyEnvironment(vei.Proxy)
	resourceLimits := k.buildResourceRequirements()

	jobInput := createJobInput{
		JobName:                 jobName,
		Namespace:               k.KubeClient.GetNamespace(),
		ContainerImage:          defaultContainerImage,
		CurlCommand:             curlCommand,
		ProxySettings:           proxySettings,
		TTLSecondsAfterFinished: defaultTTLSecondsAfterFinished,
		ActiveDeadlineSeconds:   defaultActiveDeadlineSeconds,
		BackoffLimit:            defaultBackoffLimit,
		ResourceLimits:          resourceLimits,
		Ctx:                     vei.Ctx,
	}

	err = k.createAndExecuteJob(jobInput)
	if err != nil {
		return k.Output.AddError(err)
	}

	// Collect and parse output
	err = k.collectAndParseJobOutput(vei.Ctx, jobName, vei.Probe)
	if err != nil {
		k.Output.AddError(err)
	}

	return &k.Output
}

func (k *KubeVerifier) VerifyDns(vdi verifier.VerifyDnsInput) *output.Output {
	// Placeholder implementation for DNS verification
	k.Logger.Info(vdi.Ctx, "DNS verification not yet implemented for Kubernetes verifier")
	return &k.Output
}

func (k *KubeVerifier) generateCurlCommands(egressListStr, tlsDisabledEgressListStr string, timeout time.Duration, proxyConfig proxy.ProxyConfig) (string, error) {
	// Build curlgen options
	options := &curlgen.Options{
		CaPath:          "/etc/pki/tls/certs/",
		ProxyCaPath:     "/etc/pki/tls/certs/",
		Retry:           3,
		MaxTime:         fmt.Sprintf("%.2f", timeout.Seconds()),
		NoTls:           fmt.Sprintf("%t", proxyConfig.NoTls),
		Urls:            strings.TrimSpace(egressListStr),
		TlsDisabledUrls: strings.TrimSpace(tlsDisabledEgressListStr),
	}

	// Generate the curl command using curlgen
	curlCommand, err := curlgen.GenerateString(options)
	if err != nil {
		return "", fmt.Errorf("failed to generate curl command: %w", err)
	}

	return curlCommand, nil
}

func (k *KubeVerifier) createAndExecuteJob(input createJobInput) error {
	// Build the Job specification
	job := k.buildJobSpec(input)

	// Create the Job
	k.writeDebugLogs(input.Ctx, fmt.Sprintf("Creating Job: %s", input.JobName))
	_, err := k.KubeClient.CreateJob(input.Ctx, job)
	if err != nil {
		return handledErrors.NewGenericError(fmt.Errorf("failed to create job %s: %w", input.JobName, err))
	}

	// Wait for Job completion with timeout
	k.writeDebugLogs(input.Ctx, fmt.Sprintf("Waiting for Job completion: %s", input.JobName))
	err = k.KubeClient.WaitForJobCompletion(input.Ctx, input.JobName)
	if err != nil {
		return handledErrors.NewGenericError(fmt.Errorf("job %s failed or timed out: %w", input.JobName, err))
	}

	return nil
}

func (k *KubeVerifier) buildJobSpec(input createJobInput) *batchv1.Job {
	activeDeadlineSeconds := int64(input.ActiveDeadlineSeconds)
	ptrFalse := false
	ptrTrue := true

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.JobName,
			Namespace: input.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &input.TTLSecondsAfterFinished,
			ActiveDeadlineSeconds:   &activeDeadlineSeconds,
			BackoffLimit:            &input.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &ptrFalse,
					RestartPolicy:                corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:      "osd-network-verifier",
							Image:     input.ContainerImage,
							Command:   []string{"/bin/sh", "-c"},
							Args:      []string{input.CurlCommand},
							Resources: input.ResourceLimits,
							Env:       k.buildEnvVars(input.ProxySettings),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &ptrFalse,
								ReadOnlyRootFilesystem:   &ptrTrue,
								RunAsNonRoot:             &ptrTrue,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

func (k *KubeVerifier) buildEnvVars(proxySettings map[string]string) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	for key, value := range proxySettings {
		if value != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}

	return envVars
}

func (k *KubeVerifier) collectAndParseJobOutput(ctx context.Context, jobName string, probe probes.Probe) error {
	k.writeDebugLogs(ctx, fmt.Sprintf("Collecting logs from Job: %s", jobName))

	// Get logs from the Job
	logs, err := k.KubeClient.GetJobLogs(ctx, jobName)
	if err != nil {
		return handledErrors.NewGenericError(fmt.Errorf("failed to get logs from job %s: %w", jobName, err))
	}

	// Parse the logs for probe output
	k.writeDebugLogs(ctx, fmt.Sprintf("Raw job logs:\n---\n%s\n---", logs))

	// Extract probe output between separators
	rawProbeOutput := k.parseJobLogs(logs)
	if rawProbeOutput == "" {
		return handledErrors.NewGenericError(fmt.Errorf("no valid probe output found in job logs"))
	}

	k.writeDebugLogs(ctx, fmt.Sprintf("Parsed probe output:\n---\n%s\n---", rawProbeOutput))

	// Send probe output to the Probe interface for parsing
	probe.ParseProbeOutput(false, rawProbeOutput, &k.Output)

	return nil
}

func (k *KubeVerifier) parseJobLogs(logs string) string {
	// Look for content between @NV@ separators
	lines := strings.Split(logs, "\n")
	var probeOutput []string
	capturing := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, curlgen.DefaultCurlOutputSeparator) {
			if capturing {
				// End of capture, add this line and stop
				probeOutput = append(probeOutput, line)
				break
			} else {
				// Start of capture
				capturing = true
				probeOutput = append(probeOutput, line)
			}
		} else if capturing {
			probeOutput = append(probeOutput, line)
		}
	}

	return strings.Join(probeOutput, "\n")
}

func (k *KubeVerifier) buildProxyEnvironment(proxyConfig proxy.ProxyConfig) map[string]string {
	proxyEnv := make(map[string]string)

	if proxyConfig.HttpProxy != "" {
		proxyEnv["HTTP_PROXY"] = proxyConfig.HttpProxy
		proxyEnv["http_proxy"] = proxyConfig.HttpProxy
	}

	if proxyConfig.HttpsProxy != "" {
		proxyEnv["HTTPS_PROXY"] = proxyConfig.HttpsProxy
		proxyEnv["https_proxy"] = proxyConfig.HttpsProxy
	}

	if len(proxyConfig.NoProxy) > 0 {
		noProxy := proxyConfig.NoProxyAsString()
		proxyEnv["NO_PROXY"] = noProxy
		proxyEnv["no_proxy"] = noProxy
	}

	return proxyEnv
}

func (k *KubeVerifier) buildResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
	}
}

func (k *KubeVerifier) writeDebugLogs(ctx context.Context, log string) {
	k.Output.AddDebugLogs(log)
	k.Logger.Debug(ctx, log)
}
