package curl

import (
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

// cloudMachineImageMap is a lookup table mapping VM image IDs to their
// respective cloud platforms, CPU architectures, and cloud regions. To
// access, reference cloudMachineImageMap[$CLOUD_PLATFORM][$CPU_ARCH][$REGION];
// e.g., cloudMachineImageMap[helpers.PlatformAWS][cpu.ArchX86]["us-east-1"]
// Note that GCP images are global/not region-scoped, so the region key will
// always be "*"
var cloudMachineImageMap = map[string]map[cpu.Architecture]map[string]string{
	helpers.PlatformAWS: {
		cpu.ArchX86: {
			"af-south-1":     "ami-02f0e23026ca00b2b",
			"ap-east-1":      "ami-0981ad6d75e659d4d",
			"ap-northeast-1": "ami-074c8721d73ba4b0d",
			"ap-northeast-2": "ami-0f5b024db3e7a56fa",
			"ap-northeast-3": "ami-021e87817a64eeba9",
			"ap-south-1":     "ami-05303df9e91c591e0",
			"ap-south-2":     "ami-0338eae7062a41409",
			"ap-southeast-1": "ami-0fcf6012486c77633",
			"ap-southeast-2": "ami-09195f5e7d6efc67c",
			"ap-southeast-3": "ami-02b0426356774b8e9",
			"ap-southeast-4": "ami-08c6e9a4360b81b76",
			"ca-central-1":   "ami-0e5c8ebf100cdb6aa",
			"eu-central-1":   "ami-0e860c6cf387c0af0",
			"eu-central-2":   "ami-07f1e1842c0b9b883",
			"eu-north-1":     "ami-0313cb579d11b36cf",
			"eu-south-1":     "ami-02e3b23c88d643c89",
			"eu-south-2":     "ami-0a7f7b2cbdbbb0fde",
			"eu-west-1":      "ami-012204fcf47ff7639",
			"eu-west-2":      "ami-0cefcf843c5c5d8ff",
			"eu-west-3":      "ami-0eb6b1f8ce56c5021",
			"me-central-1":   "ami-0bbb6a021b8bb7e62",
			"me-south-1":     "ami-077846abfedb9d525",
			"sa-east-1":      "ami-0f057eaf639455796",
			"us-east-1":      "ami-0b7c8a5ca88a68fcb",
			"us-east-2":      "ami-0051b519afc5528a1",
			"us-west-1":      "ami-096b3a8ac97ec26d0",
			"us-west-2":      "ami-0e6f3bbd0f97807e8",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-TODO",
			"ap-east-1":      "ami-TODO",
			"ap-northeast-1": "ami-TODO",
			"ap-northeast-2": "ami-TODO",
			"ap-northeast-3": "ami-TODO",
			"ap-south-1":     "ami-TODO",
			"ap-south-2":     "ami-TODO",
			"ap-southeast-1": "ami-TODO",
			"ap-southeast-2": "ami-TODO",
			"ap-southeast-3": "ami-TODO",
			"ap-southeast-4": "ami-TODO",
			"ca-central-1":   "ami-TODO",
			"eu-central-1":   "ami-TODO",
			"eu-central-2":   "ami-TODO",
			"eu-north-1":     "ami-TODO",
			"eu-south-1":     "ami-TODO",
			"eu-south-2":     "ami-TODO",
			"eu-west-1":      "ami-TODO",
			"eu-west-2":      "ami-TODO",
			"eu-west-3":      "ami-TODO",
			"me-central-1":   "ami-TODO",
			"me-south-1":     "ami-TODO",
			"sa-east-1":      "ami-TODO",
			"us-east-1":      "ami-TODO",
			"us-east-2":      "ami-TODO",
			"us-west-1":      "ami-TODO",
			"us-west-2":      "ami-TODO",
		},
	},
	// See function docstring's note on GCP; tl;dr: deepest key should be "*"
	helpers.PlatformGCP: {
		cpu.ArchX86: {
			"*": "rhel-9",
		},
		cpu.ArchARM: {
			"*": "rhel-9-arm64",
		},
	},
}
