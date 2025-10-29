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
			"af-south-1":     "ami-0ec4258073990d047",
			"ap-east-1":      "ami-0ddbea1765fc588dd",
			"ap-northeast-1": "ami-0a1172118e5d94307",
			"ap-northeast-2": "ami-0f49e409ce3644f0b",
			"ap-northeast-3": "ami-0da842df4ab63f36b",
			"ap-south-1":     "ami-038881923394dbec4",
			"ap-south-2":     "ami-0bde283ee16bb6f57",
			"ap-southeast-1": "ami-0a620d1500051d462",
			"ap-southeast-2": "ami-06345d7eb58811580",
			"ap-southeast-3": "ami-08e25307b4ffe983b",
			"ap-southeast-4": "ami-0e620378ec2f6808d",
			"ap-southeast-5": "ami-0e2f96f66671fa763",
			"ap-southeast-6": "ami-0939f607da6dcb54f",
			"ap-southeast-7": "ami-033129d0e220b26cc",
			"ca-central-1":   "ami-06505bae1e3920b76",
			"ca-west-1":      "ami-0aed7134a0f17a34b",
			"eu-central-1":   "ami-080418db5fcbe1891",
			"eu-central-2":   "ami-070bf99d5467ed89a",
			"eu-north-1":     "ami-01740f3a866e26fd0",
			"eu-south-1":     "ami-084ab4ab27a60ddc3",
			"eu-south-2":     "ami-0cc1172245179435f",
			"eu-west-1":      "ami-02cc4b70b5dd07471",
			"eu-west-2":      "ami-0f40f5d49016d7b9f",
			"eu-west-3":      "ami-0f6bb50b30e2a00ce",
			"il-central-1":   "ami-07b5eb823681c0e3a",
			"me-central-1":   "ami-02e8aba1933525746",
			"me-south-1":     "ami-049bed9f1bb06e2cb",
			"mx-central-1":   "ami-07c7a3c805e898528",
			"sa-east-1":      "ami-041ac006a7de1260c",
			"us-east-1":      "ami-0f1c3fe3fab901753",
			"us-east-2":      "ami-0c03ff48d83bfa566",
			"us-west-1":      "ami-01f23058474a9f3dd",
			"us-west-2":      "ami-0eaeb25dfb8dad6f5",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-09331da5555b447bb",
			"ap-east-1":      "ami-049b7f14f784e8fe5",
			"ap-northeast-1": "ami-02cbe6c39bd6d2a2c",
			"ap-northeast-2": "ami-0b17f421cd558077f",
			"ap-northeast-3": "ami-0e77eabe5db69996c",
			"ap-south-1":     "ami-05e9c3ffca82c0ab6",
			"ap-south-2":     "ami-09b77194da051c539",
			"ap-southeast-1": "ami-0bbe33feb9248bdc5",
			"ap-southeast-2": "ami-087959cce12178d09",
			"ap-southeast-3": "ami-09bff3d44e55a8bb5",
			"ap-southeast-4": "ami-061fb73c6640ad5b3",
			"ap-southeast-5": "ami-00548f7e0f86bfaf8",
			"ap-southeast-6": "ami-0d3108fa28d6eb51a",
			"ap-southeast-7": "ami-034854a3b38a18c26",
			"ca-central-1":   "ami-0fd06f1e5e898b919",
			"ca-west-1":      "ami-07df6e67c2c22c944",
			"eu-central-1":   "ami-04196742fde29e9ef",
			"eu-central-2":   "ami-0fa4e5aca1cda4a86",
			"eu-north-1":     "ami-071b7778fd83e3d6c",
			"eu-south-1":     "ami-00033a961e080ee3c",
			"eu-south-2":     "ami-0be555cdf8cefa972",
			"eu-west-1":      "ami-09842165b26054486",
			"eu-west-2":      "ami-01cfeea82ba3e1ccd",
			"eu-west-3":      "ami-06cc703bcb9ba89d0",
			"il-central-1":   "ami-0699cd84a52190c6d",
			"me-central-1":   "ami-0a0f7753ad304dd94",
			"me-south-1":     "ami-015aa540c9342c53e",
			"mx-central-1":   "ami-0fc643c95274c669a",
			"sa-east-1":      "ami-0b529fa606110b90b",
			"us-east-1":      "ami-03bc612e8d064b016",
			"us-east-2":      "ami-0d82a10fedbb2237d",
			"us-west-1":      "ami-079aa45b1e48e3614",
			"us-west-2":      "ami-084026f634b808f4d",
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
