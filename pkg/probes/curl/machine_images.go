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
			"af-south-1":     "ami-0f452a231fc097fcb",
			"ap-east-1":      "ami-0cc8701f607761370",
			"ap-northeast-1": "ami-0df99a04742ab1578",
			"ap-northeast-2": "ami-0e8d2c20b8779227d",
			"ap-northeast-3": "ami-0c169d50e75cc4de5",
			"ap-south-1":     "ami-0c5341420fdf04739",
			"ap-south-2":     "ami-070f82caecf16100d",
			"ap-southeast-1": "ami-0dd8ef70ff1606b5b",
			"ap-southeast-2": "ami-0306976dcfeb71c0e",
			"ap-southeast-3": "ami-05723d15bafef6562",
			"ap-southeast-4": "ami-0eeb016433ba5f9d8",
			"ap-southeast-5": "ami-031f1e42ea38585f4",
			"ap-southeast-6": "ami-064af18c88736b4d1",
			"ap-southeast-7": "ami-0b310965a175f44d2",
			"ca-central-1":   "ami-074d29ace9d434d23",
			"ca-west-1":      "ami-0eaa120e68026a155",
			"eu-central-1":   "ami-0bbcdbbbac1e7ab48",
			"eu-central-2":   "ami-0b38ca4b411bb400a",
			"eu-north-1":     "ami-0d6be67f06adb134e",
			"eu-south-1":     "ami-0857dce8b48b2110c",
			"eu-south-2":     "ami-0574a95a0c0b3ab20",
			"eu-west-1":      "ami-0608965b64d9d175e",
			"eu-west-2":      "ami-00750b9966c551c4b",
			"eu-west-3":      "ami-027cd1ef039427aa3",
			"il-central-1":   "ami-0580f6aa68aa91786",
			"mx-central-1":   "ami-0419ebd26f164310d",
			"sa-east-1":      "ami-00f3915624345d134",
			"us-east-1":      "ami-0a75c0bb5dd7ff3c3",
			"us-east-2":      "ami-0e5845e9b52df1e16",
			"us-west-1":      "ami-0a0aac0b5a458f105",
			"us-west-2":      "ami-08ac43c3767375853",
		},
		cpu.ArchARM: {
			"af-south-1":     "ami-04cb1004d125a5e6d",
			"ap-east-1":      "ami-0018d5184805ea1c6",
			"ap-northeast-1": "ami-08b5d4c6bdfea048f",
			"ap-northeast-2": "ami-02a106e43a3734b44",
			"ap-northeast-3": "ami-03ab9cf36cf060f7f",
			"ap-south-1":     "ami-04df94fc729a63983",
			"ap-south-2":     "ami-0ac5c90df3f514e52",
			"ap-southeast-1": "ami-0533b8d6d0a255a3b",
			"ap-southeast-2": "ami-0a8db5f3ff3d76a25",
			"ap-southeast-3": "ami-0296f255ae96c0a6d",
			"ap-southeast-4": "ami-099e692a16ee0689d",
			"ap-southeast-5": "ami-0c2138debdffb538c",
			"ap-southeast-6": "ami-07ce94c7dd614bb79",
			"ap-southeast-7": "ami-01d15f2ed28c35621",
			"ca-central-1":   "ami-081c4dc33f7cc61e0",
			"ca-west-1":      "ami-038d41754361a075c",
			"eu-central-1":   "ami-093479107906966cb",
			"eu-central-2":   "ami-01086fc63e9dd2a96",
			"eu-north-1":     "ami-0b0a7e638b4746b6a",
			"eu-south-1":     "ami-0bfca626643a333db",
			"eu-south-2":     "ami-077a85b9ed42dc356",
			"eu-west-1":      "ami-02d46124fec958d0a",
			"eu-west-2":      "ami-08dd84c08ef9a6e91",
			"eu-west-3":      "ami-02aa098087ca27b17",
			"il-central-1":   "ami-069cac22773acc337",
			"mx-central-1":   "ami-0ae01faf1789649c5",
			"sa-east-1":      "ami-079df9584ae729779",
			"us-east-1":      "ami-02ede7286b05219e6",
			"us-east-2":      "ami-0671504ca0f06408e",
			"us-west-1":      "ami-05d96e7a17d88a591",
			"us-west-2":      "ami-070f44ae75d57e6e5",
		},
	},
	cloud.AWSGovCloudClassic: {
		cpu.ArchX86: {
			"us-gov-west-1": "ami-030d1cf861950bc2b",
			"us-gov-east-1": "ami-02b32ce96992f4454",
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
