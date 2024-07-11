package cpu

import "github.com/openshift/osd-network-verifier/pkg/helpers"

// defaultInstanceTypes is a lookup table mapping cloud instance/machine types to their
// respective cloud platforms and CPU architectures. See Architecture.DefaultInstanceType()
// for more details
var defaultInstanceTypes = map[string]map[Architecture]string{
	helpers.PlatformAWS: {
		ArchX86: "t3.micro",
		ArchARM: "t4g.micro",
	},
	helpers.PlatformGCP: {
		ArchX86: "e2-micro",
		ArchARM: "t2a-standard-1",
	},
}
