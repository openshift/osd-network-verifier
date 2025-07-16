package kube

import (
	"fmt"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/kube"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"k8s.io/client-go/kubernetes"
)

type KubeVerifier struct {
	KubeClient kube.Client
	Logger     ocmlog.Logger
	Output     output.Output
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
