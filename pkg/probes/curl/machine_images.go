package curl

import (
	cloud "github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
)

// cloudMachineImageMap is a lookup table mapping VM image IDs to their
// respective cloud platforms, CPU architectures, and cloud regions. To
// access, reference cloudMachineImageMap[$CLOUD_PLATFORM][$CPU_ARCH][$REGION];
// e.g., cloudMachineImageMap[helpers.PlatformAWS][cpu.ArchX86]["us-east-1"]
// Note that GCP images are global/not region-scoped, so the region key will
// always be "*"
var cloudMachineImageMap = map[cloud.Platform]map[cpu.Architecture]map[string]string{
	cloud.AWSClassic: {
		cpu.ArchX86: {
			"af-south-1":     "ami-0974db472280394e1",
			"ap-east-1":      "ami-03a4cbb657e8ea739",
			"ap-northeast-1": "ami-0ae8ffa855c8177b2",
			"ap-northeast-2": "ami-0358624b50042d409",
			"ap-northeast-3": "ami-0a22b51e29ed2537d",
			"ap-south-1":     "ami-021518296223d8409",
			"ap-south-2":     "ami-0578b48ff45674c03",
			"ap-southeast-1": "ami-06f2f752d3eff1eb0",
			"ap-southeast-2": "ami-05cd5c2cc4560fa91",
			"ap-southeast-3": "ami-08295e5242802c048",
			"ap-southeast-4": "ami-0ed59a31df7106e23",
			"ca-central-1":   "ami-0f25b40eef43ae2dc",
			"eu-central-1":   "ami-06b1db3b977a8ebba",
			"eu-central-2":   "ami-00495c1da61921eaf",
			"eu-north-1":     "ami-0ddcbf7d5012478df",
			"eu-south-1":     "ami-01a0800e6a64928dd",
			"eu-south-2":     "ami-055ac2df903649cfa",
			"eu-west-1":      "ami-0a9798d1f71d9b061",
			"eu-west-2":      "ami-0990e877d04571b01",
			"eu-west-3":      "ami-00bb02874f2992977",
			"me-central-1":   "ami-0b8e8cf82f251153d",
			"me-south-1":     "ami-0bf4719fb9243a28c",
			"sa-east-1":      "ami-0a3b275687ba03505",
			"us-east-1":      "ami-01af63b960f312130",
			"us-east-2":      "ami-0a99fdad384edf476",
			"us-west-1":      "ami-031e00adaa17c14d7",
			"us-west-2":      "ami-042d4c3472784287c",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-00d4805ffb126b3ca",
			"ap-east-1":      "ami-020d8309cf51a242a",
			"ap-northeast-1": "ami-05b332269c4b20121",
			"ap-northeast-2": "ami-0091a16de450e462e",
			"ap-northeast-3": "ami-019ffab309e1b6c23",
			"ap-south-1":     "ami-0cb40f0300112ca3d",
			"ap-south-2":     "ami-077177118385f0593",
			"ap-southeast-1": "ami-06ddccfbc22a4520c",
			"ap-southeast-2": "ami-0c1ef205b1eb9ef41",
			"ap-southeast-3": "ami-030597601ae5e1291",
			"ap-southeast-4": "ami-0996cbf948342585c",
			"ca-central-1":   "ami-0bd776ced443071f9",
			"eu-central-1":   "ami-02c801832c20f84b7",
			"eu-central-2":   "ami-053028d28f4a1f190",
			"eu-north-1":     "ami-0a5e85ea40c04d448",
			"eu-south-1":     "ami-0fd6578401210bf93",
			"eu-south-2":     "ami-0b2707b92b1b22d2b",
			"eu-west-1":      "ami-0b1b39c8901999936",
			"eu-west-2":      "ami-0adf094d79cff8702",
			"eu-west-3":      "ami-07332a0b8cc1e3873",
			"me-central-1":   "ami-0b14c8325b89ba86b",
			"me-south-1":     "ami-046b6ff11c35b5dcc",
			"sa-east-1":      "ami-0d7270b041533f78b",
			"us-east-1":      "ami-0d59c56924e928c97",
			"us-east-2":      "ami-030bc769bda1b3560",
			"us-west-1":      "ami-0b389db946d3728d5",
			"us-west-2":      "ami-0c9c6f4cd8c80b431",
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
