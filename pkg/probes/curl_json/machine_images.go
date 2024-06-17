package curl_json

import "github.com/openshift/osd-network-verifier/pkg/helpers"

// cloudMachineImageMap is a lookup table mapping VM image IDs to their
// respective cloud platforms, CPU architectures, and cloud regions. To
// access, reference cloudMachineImageMap[$CLOUD_PLATFORM][$CPU_ARCH][$REGION];
// e.g., cloudMachineImageMap[helpers.PlatformAWS][helpers.ArchX86]["us-east-1"]
// Note that GCP images are global/not region-scoped, so the region key will
// always be "*"
var cloudMachineImageMap = map[string]map[string]map[string]string{
	helpers.PlatformAWS: {
		helpers.ArchX86: {
			"af-south-1":     "ami-TODO",
			"ap-east-1":      "ami-TODO",
			"ap-northeast-1": "ami-0a3299a47e8a9111b",
			"ap-northeast-2": "ami-09810edf6c4f708dc",
			"ap-northeast-3": "ami-0905964fe4d783b7d",
			"ap-south-1":     "ami-04708942c263d8190",
			"ap-south-2":     "ami-TODO",
			"ap-southeast-1": "ami-0371cc1c8b8e24fdc",
			"ap-southeast-2": "ami-0ade3fd7d152f84df",
			"ap-southeast-3": "ami-TODO",
			"ap-southeast-4": "ami-TODO",
			"ca-central-1":   "ami-04978032a1284973a",
			"eu-central-1":   "ami-076433a70aba7f25d",
			"eu-central-2":   "ami-TODO",
			"eu-north-1":     "ami-01d565a5f2da42e6f",
			"eu-south-1":     "ami-TODO",
			"eu-south-2":     "ami-TODO",
			"eu-west-1":      "ami-049b0abf844cab8d7",
			"eu-west-2":      "ami-08c3913593117726b",
			"eu-west-3":      "ami-0bd23a7080ec75f4d",
			"me-central-1":   "ami-TODO",
			"me-south-1":     "ami-TODO",
			"sa-east-1":      "ami-00b45eebb277341fe",
			"us-east-1":      "ami-023c11a32b0207432",
			"us-east-2":      "ami-0ef50c2b2eb330511",
			"us-west-1":      "ami-0e534e4c6bae7faf7",
			"us-west-2":      "ami-04b4d3355a2e2a403",
		},
		helpers.ArchARM: {
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
		helpers.ArchX86: {
			"*": "rhel-9",
		},
		helpers.ArchARM: {
			"*": "rhel-9-arm64",
		},
	},
}
