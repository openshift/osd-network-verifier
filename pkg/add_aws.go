package cloudclient

import "github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"

func init() {
	Register(
		aws.ClientIdentifier,
		produceAWS,
	)
}

func produceAWS() CloudClient {
	cli, err := aws.NewClient()
	if err != nil {
		panic(err)
	}

	return cli
}
