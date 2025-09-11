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
			"af-south-1":     "ami-03b3318d821456042",
			"ap-east-1":      "ami-0430228fe65ed7061",
			"ap-northeast-1": "ami-02c871c5862fc84c3",
			"ap-northeast-2": "ami-07cce669b7faa2550",
			"ap-northeast-3": "ami-036f1f4f19df96ec8",
			"ap-south-1":     "ami-04dbcb2c9412cda21",
			"ap-south-2":     "ami-019242deff93615db",
			"ap-southeast-1": "ami-0cc6d8f93aade7af4",
			"ap-southeast-2": "ami-0c3345b0a028769f2",
			"ap-southeast-3": "ami-0096ecc3657fdceb3",
			"ap-southeast-4": "ami-041d59f4d995b7de1",
			"ap-southeast-5": "ami-0fca5b6cd808536a7",
			"ap-southeast-7": "ami-0dac6ff30881f8070",
			"ca-central-1":   "ami-0444c53df356a420d",
			"ca-west-1":      "ami-093fb5a2263421324",
			"eu-central-1":   "ami-049930a3cb6f58549",
			"eu-central-2":   "ami-0647f36ce842fc233",
			"eu-north-1":     "ami-08cef705e10ca44df",
			"eu-south-1":     "ami-0907217e48a0ee937",
			"eu-south-2":     "ami-0a35c1e67bf60a28f",
			"eu-west-1":      "ami-08745c253d1a6a9ed",
			"eu-west-2":      "ami-08702dfd7334a7ef0",
			"eu-west-3":      "ami-06e89a37fe3ec13f9",
			"il-central-1":   "ami-06043668f6fa56e85",
			"me-central-1":   "ami-073bbb720eaeeef42",
			"me-south-1":     "ami-004c09cb0c2c8a172",
			"mx-central-1":   "ami-056e992a0a761b1dc",
			"sa-east-1":      "ami-0d78f8c8170c29ffb",
			"us-east-1":      "ami-0136fb593fc7d8b9a",
			"us-east-2":      "ami-0d2706738cfcee73d",
			"us-west-1":      "ami-094e9452686ad73c8",
			"us-west-2":      "ami-0ebffe604f8cc4d1c",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-0f5c05ad89a078e18",
			"ap-east-1":      "ami-0aa909e350eb9f8b8",
			"ap-northeast-1": "ami-0f8e41f66583f23a3",
			"ap-northeast-2": "ami-0a3232788f241a1f9",
			"ap-northeast-3": "ami-0965a9b0f822df216",
			"ap-south-1":     "ami-0b1a6b9dde7f8eb62",
			"ap-south-2":     "ami-0e2748a376d8502ad",
			"ap-southeast-1": "ami-05d6ed6d598474e99",
			"ap-southeast-2": "ami-0f16401cba5a149a2",
			"ap-southeast-3": "ami-0f2624b4b7bb25424",
			"ap-southeast-4": "ami-0f15deffa346c64f4",
			"ap-southeast-5": "ami-0e5ae0d044bd14c0c",
			"ap-southeast-7": "ami-040cae318240d626e",
			"ca-central-1":   "ami-02311072b9fdbcfe4",
			"ca-west-1":      "ami-01e5e082537b8a7c1",
			"eu-central-1":   "ami-0a29bc65059a49879",
			"eu-central-2":   "ami-0f32ddaea8aa1be69",
			"eu-north-1":     "ami-0c2002c71382e975e",
			"eu-south-1":     "ami-07bd39ff178de4004",
			"eu-south-2":     "ami-0a6819d438f29f402",
			"eu-west-1":      "ami-07518c127652d9e96",
			"eu-west-2":      "ami-084ce9a7851eed82d",
			"eu-west-3":      "ami-01d2d5880c90deede",
			"il-central-1":   "ami-0f4fe9fde9067711a",
			"me-central-1":   "ami-080662d3a0fec5599",
			"me-south-1":     "ami-00020b7da57013ab4",
			"mx-central-1":   "ami-0ac30785e9e721dab",
			"sa-east-1":      "ami-085c3db6f55f607f5",
			"us-east-1":      "ami-0525f5292d6e0dee3",
			"us-east-2":      "ami-0b47331547192e6fe",
			"us-west-1":      "ami-01a9e146fb91281cc",
			"us-west-2":      "ami-036a99a592001ad8d",
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
