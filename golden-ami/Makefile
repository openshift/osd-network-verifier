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
