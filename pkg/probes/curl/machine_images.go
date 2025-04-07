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
			"af-south-1":     "ami-09450d71eb76c972e",
			"ap-east-1":      "ami-05331c4362de63b38",
			"ap-northeast-1": "ami-048677bbbe474029b",
			"ap-northeast-2": "ami-0bc098369bb6e8ae6",
			"ap-northeast-3": "ami-01a035f3283415dc5",
			"ap-south-1":     "ami-0b6165cccdaebffe1",
			"ap-south-2":     "ami-0ac918f40c9f2be1e",
			"ap-southeast-1": "ami-05cfb9a8f41e06073",
			"ap-southeast-2": "ami-08b947fc447dd1b5b",
			"ap-southeast-3": "ami-03f37f55595be57db",
			"ap-southeast-4": "ami-0fcedbc61601ea5c4",
			"ap-southeast-5": "ami-07dadbd8631dcf109",
			"ca-central-1":   "ami-09ed7b821be1a37fd",
			"ca-west-1":      "ami-0427a1065bb2aa19d",
			"eu-central-1":   "ami-0ce424362ff969e28",
			"eu-central-2":   "ami-020ebdb65aa36a76f",
			"eu-north-1":     "ami-002f4bbc181fc9a1f",
			"eu-south-1":     "ami-0f36bbcb07706d8a2",
			"eu-south-2":     "ami-0c63550e7ddf76600",
			"eu-west-1":      "ami-0eac6cff9c0eb67b4",
			"eu-west-2":      "ami-0a4555c24578ed3ae",
			"eu-west-3":      "ami-0d34c6865aa7717be",
			"il-central-1":   "ami-05b5a08273fec619b",
			"me-central-1":   "ami-0d091e9b934be542f",
			"me-south-1":     "ami-06c555baf17a671f4",
			"sa-east-1":      "ami-08b073294ada15eff",
			"us-east-1":      "ami-024cb5f61142db5da",
			"us-east-2":      "ami-040448152c2fa56c4",
			"us-west-1":      "ami-0cc66682c05efd9dc",
			"us-west-2":      "ami-090706d0f72df52be",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-09a0a4f0ee1e73d12",
			"ap-east-1":      "ami-0b4f33b5a9f4e1e6e",
			"ap-northeast-1": "ami-0f993727a4789bf1e",
			"ap-northeast-2": "ami-07a1607192ce6dced",
			"ap-northeast-3": "ami-091173299d41d10b9",
			"ap-south-1":     "ami-099eb9342e8b2fcce",
			"ap-south-2":     "ami-0c135889269df8d81",
			"ap-southeast-1": "ami-0c0fc3b1d7f660915",
			"ap-southeast-2": "ami-0c855ccb61921ebe1",
			"ap-southeast-3": "ami-09fb0e55835d346e3",
			"ap-southeast-4": "ami-0f9678f5718b2dae6",
			"ap-southeast-5": "ami-0a0afef667557be5f",
			"ca-central-1":   "ami-007a2616001dc46fc",
			"ca-west-1":      "ami-0e4d5134761407698",
			"eu-central-1":   "ami-02ab75d21f1b22abe",
			"eu-central-2":   "ami-0513fc1b58b355cc8",
			"eu-north-1":     "ami-074f0baf07ff3672d",
			"eu-south-1":     "ami-02c0d0bb6667b7204",
			"eu-south-2":     "ami-04a6afc134589ce57",
			"eu-west-1":      "ami-068853e262253abd6",
			"eu-west-2":      "ami-07d78e8cfade0f4a2",
			"eu-west-3":      "ami-091cff70c34d4bcc6",
			"il-central-1":   "ami-0358d593bd3f22fa0",
			"me-central-1":   "ami-03b37c98249e5a43b",
			"me-south-1":     "ami-046e9602693d97fe7",
			"sa-east-1":      "ami-071b3319ae50b3e80",
			"us-east-1":      "ami-08f8be531c4de968d",
			"us-east-2":      "ami-004fe2a1faa73e370",
			"us-west-1":      "ami-0a8798f603e1f35b8",
			"us-west-2":      "ami-063d3d7aa1afc7503",
		},
	},
	// See function docstring's note on GCP; tl;dr: deepest key should be "*"
	cloud.GCPClassic: {
		cpu.ArchX86: {
			"*": "rhel-9-v20240709",
		},
		cpu.ArchARM: {
			"*": "rhel-9-arm64-v20240709",
		},
	},
}
