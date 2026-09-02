# Legacy AMI targets: these build the network-validator container locally but do
# NOT push it to a registry. To use the legacy AMI workflow, you must manually
# push the container image and pass its URI to run.sh. CI does not use these
# targets; it uses build-all-amis-ci (RHEL repackage) instead.
.PHONY: build-and-publish-docker-and-ami
build-and-publish-docker-and-ami: build-docker
	./run.sh ${IMAGE_URI}

.PHONY: build-docker
build-docker:
	docker build -t osd-network-verifier -f build/Dockerfile .

.PHONY: build-docker-and-ami
build-docker-and-ami: build-docker
	./run.sh

.PHONY: copy-rhel-ami
copy-rhel-ami:
	./copy-rhel-ami.sh $(arch)

.PHONY: copy-rhel-ami-x86_64
copy-rhel-ami-x86_64:
	./copy-rhel-ami.sh x86_64

.PHONY: copy-rhel-ami-arm64
copy-rhel-ami-arm64:
	./copy-rhel-ami.sh arm64

.PHONY: build-all-amis-ci
build-all-amis-ci: copy-rhel-ami-arm64 copy-rhel-ami-x86_64
