package cloudclient

import "github.com/openshift/osd-network-verifier/pkg/cloudclient/gcp"

func init() {
	Register(
		gcp.ClientIdentifier,
		produceGCP,
	)
}

func produceGCP() CloudClient {
	cli, err := gcp.NewClient()
	if err != nil {
		panic(err)
	}

	return cli
}
