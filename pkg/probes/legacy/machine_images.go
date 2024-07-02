package legacy

import "github.com/openshift/osd-network-verifier/pkg/helpers"

// cloudMachineImageMap is a lookup table mapping VM image IDs to their
// respective cloud platforms, CPU architectures, and cloud regions. To
// access, reference cloudMachineImageMap[$CLOUD_PLATFORM][$CPU_ARCH][$REGION];
// e.g., cloudMachineImageMap[helpers.PlatformAWS][helpers.ArchX86]["us-east-1"]
// Note that the legacy probe only has ever supported X86 on AWS
var cloudMachineImageMap = map[string]map[string]map[string]string{
	helpers.PlatformAWS: {
		helpers.ArchX86: {
			"af-south-1":     "ami-082888538e0d5ab6f",
			"ap-east-1":      "ami-0e8a82f83fd6c4671",
			"ap-northeast-1": "ami-0e46d69767db39b8d",
			"ap-northeast-2": "ami-091b8cb907bfb2b56",
			"ap-northeast-3": "ami-02cabb26586b45336",
			"ap-south-1":     "ami-00f693020f4aed8ce",
			"ap-south-2":     "ami-056ca61a0e723593c",
			"ap-southeast-1": "ami-0a63318003f651f49",
			"ap-southeast-2": "ami-05bd8629c45460482",
			"ap-southeast-3": "ami-0e6a3cc1f68092eba",
			"ap-southeast-4": "ami-07f057574ec80ec81",
			"ca-central-1":   "ami-06e8f78cab9e62a58",
			"eu-central-1":   "ami-08a506dd4bc126ae5",
			"eu-central-2":   "ami-0ad12e73811a04164",
			"eu-north-1":     "ami-0c5c0dc42df65c3c1",
			"eu-south-1":     "ami-0cc908758c212fe11",
			"eu-south-2":     "ami-0cce68aa0356ae420",
			"eu-west-1":      "ami-0913e4ee0fa91649a",
			"eu-west-2":      "ami-0a951043c6078f378",
			"eu-west-3":      "ami-058406cc445b09762",
			"me-central-1":   "ami-095b8831ceb37f108",
			"me-south-1":     "ami-00624346da9330d80",
			"sa-east-1":      "ami-0d1958d70a8d683e2",
			"us-east-1":      "ami-022e75a8d568b7d0b",
			"us-east-2":      "ami-0b68b178fecfcbe51",
			"us-west-1":      "ami-087c2ca9f260a820b",
			"us-west-2":      "ami-0c03998bcb7c924f9",
		},
	},
}
