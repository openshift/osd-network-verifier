package main

# Source AMI Filtering Tests
deny[msg] {
	filters := input[_][_][_].source_ami_filter
	virtualization := filters[_]["virtualization-type"]
	virtualization != "hvm"
	msg = sprintf("virtualization type should be hvm, but got: %s", [virtualization])
}

deny[msg] {
	filters := input[_][_][_].source_ami_filter
	root_device_type := filters[_]["root-device-type"]
	root_device_type != "ebs"
	msg = sprintf("root-device-type type should be ebs, but got: %s", [root_device_type])
}

# Variable Tests
## Check Image Owner is properly set
deny[msg] {
	owner_of_image := "309956199498"
	imageID := input["variable"].image_owner["default"]
	imageID != owner_of_image
	msg = sprintf("image ID should be constant(%s) got: %s", [owner_of_image,imageID])
}

## Check base ami filter is properly set
deny[msg] {
	base_image_filter := "RHEL-8.4*-x86_64-*"
	base_image := input["variable"].base_image_filter["default"]
	base_image != base_image_filter
	msg = sprintf("base ami filtering should be constant(%s) got: %s", [base_image_filter,base_image])
}