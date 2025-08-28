package curl

import (
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
)

// cloudMachineImageMap is a lookup table mapping VM image IDs to their
// respective cloud platforms, CPU architectures, and cloud regions. To
// access, reference cloudMachineImageMap[$CLOUD_PLATFORM][$CPU_ARCH][$REGION];
// e.g., cloudMachineImageMap[cloud.PlatformAWS][cpu.ArchX86]["us-east-1"]
// Note that GCP images are global/not region-scoped, so the region key will
// always be "*"
var cloudMachineImageMap = map[cloud.Platform]map[cpu.Architecture]map[string]string{
	cloud.AWSClassic: {
		cpu.ArchX86: {
			"af-south-1":     "ami-0f9feb06cfc598320",
			"ap-east-1":      "ami-0515b4a48dc3c3afe",
			"ap-northeast-1": "ami-09fc623bda798984b",
			"ap-northeast-2": "ami-03805231f9efc9a79",
			"ap-northeast-3": "ami-0e1765539bee9a1cb",
			"ap-south-1":     "ami-0c3faf3b63a3edcd6",
			"ap-south-2":     "ami-093de97147ff2a030",
			"ap-southeast-1": "ami-03b37e5946deae09a",
			"ap-southeast-2": "ami-009dc818b56066d45",
			"ap-southeast-3": "ami-03e8d7377ad866cb4",
			"ap-southeast-4": "ami-026eaf3007c5f497d",
			"ap-southeast-5": "ami-0bd241422952a2647",
			"ap-southeast-7": "ami-051447c5bfb3fb42e",
			"ca-central-1":   "ami-06ff2ec5ba87f220d",
			"ca-west-1":      "ami-08d88e4de4269acbb",
			"eu-central-1":   "ami-020d8593f4901c4f4",
			"eu-central-2":   "ami-06232baee6728e715",
			"eu-north-1":     "ami-0f8f6a687c607a550",
			"eu-south-1":     "ami-0e570282886fb4e3d",
			"eu-south-2":     "ami-0c8f40bf3451aebab",
			"eu-west-1":      "ami-02eab3db24f702c02",
			"eu-west-2":      "ami-040c9f5fcf70728f5",
			"eu-west-3":      "ami-05b2450f529a23e76",
			"il-central-1":   "ami-0a04cb7cab0cc532b",
			"me-central-1":   "ami-0e4796e363bbe4abd",
			"me-south-1":     "ami-045a0b951f0924c50",
			"sa-east-1":      "ami-0d60ee1dff81c19d1",
			"us-east-1":      "ami-07151a44e78894315",
			"us-east-2":      "ami-08b7bf41ea649338d",
			"us-west-1":      "ami-0727bbc3fbc602855",
			"us-west-2":      "ami-0b764bec9adcff1d6",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-0767949f84485b400",
			"ap-east-1":      "ami-03dbcbcada0cc24fa",
			"ap-northeast-1": "ami-0681e73990348937b",
			"ap-northeast-2": "ami-0298132ad171af0ec",
			"ap-northeast-3": "ami-0b13f120577ca513c",
			"ap-south-1":     "ami-03c37b2aedeec985a",
			"ap-south-2":     "ami-048053dc208073138",
			"ap-southeast-1": "ami-089a602b2d253eefc",
			"ap-southeast-2": "ami-0677568501a848ef4",
			"ap-southeast-3": "ami-066e38997834d0922",
			"ap-southeast-4": "ami-00956d42c4f866224",
			"ap-southeast-5": "ami-0d9cdbdad9ef342c0",
			"ap-southeast-7": "ami-00cd89802e62921c7",
			"ca-central-1":   "ami-07540e57823878b4a",
			"ca-west-1":      "ami-01b762faaf798b3d3",
			"eu-central-1":   "ami-0785438c63a831450",
			"eu-central-2":   "ami-06bb99dfd403348f5",
			"eu-north-1":     "ami-0a3157f197e010284",
			"eu-south-1":     "ami-0a6316e72c7be73bb",
			"eu-south-2":     "ami-0acd40c36b226852c",
			"eu-west-1":      "ami-023f2c9eb36dcf2bd",
			"eu-west-2":      "ami-07a9e4a1723312e0d",
			"eu-west-3":      "ami-0be723df2ea01e1ec",
			"il-central-1":   "ami-0b3c22575d09fc4ab",
			"me-central-1":   "ami-0bafa68678ef637a5",
			"me-south-1":     "ami-0875d9ac0ef2da83f",
			"sa-east-1":      "ami-091892ff4a4bf2890",
			"us-east-1":      "ami-0f5185c205254922b",
			"us-east-2":      "ami-06ffa3a3e230fd8ee",
			"us-west-1":      "ami-0fbbc02ac68e5e91e",
			"us-west-2":      "ami-0806cf404125d0f39",
		},
	},
	// See function docstring's note on GCP; tl;dr: deepest key should be "*"
	cloud.GCPClassic: {
		cpu.ArchX86: {
			"*": "rhel-9-v20250709",
		},
		cpu.ArchARM: {
			"*": "rhel-9-arm64-v20250709",
		},
	},
}
