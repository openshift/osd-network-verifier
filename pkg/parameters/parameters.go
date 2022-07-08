// argument command struct based on ocm cli
// https://github.com/openshift-online/ocm-cli/blob/main/pkg/arguments/arguments.go

package parameters

type ValidateEgress struct {
	VpcSubnetID string
}

type ValidateDns struct {
	VpcId string
}

type ValidateByoVpc struct {
	//	todo
}
