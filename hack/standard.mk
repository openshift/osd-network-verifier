# Validate variables in project.mk exist
ifndef IMAGE_REGISTRY
$(error IMAGE_REGISTRY is not set; check project.mk file)
endif
ifndef IMAGE_REPOSITORY
$(error IMAGE_REPOSITORY is not set; check project.mk file)
endif
ifndef IMAGE_NAME
$(error IMAGE_NAME is not set; check project.mk file)
endif

# Generate version and tag information from inputs
CURRENT_COMMIT=$(shell git rev-parse --short=7 HEAD)
INITIAL_COMMIT=$(shell git rev-list --max-parents=0 HEAD)
COMMIT_NUMBER=$(shell git rev-list $(INITIAL_COMMIT)..HEAD --count)
IMAGE_VERSION=$(COMMIT_NUMBER)-$(CURRENT_COMMIT)

IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME)
IMAGE_URI_VERSION=$(IMAGE_URI):v$(IMAGE_VERSION)
IMAGE_URI_LATEST=$(IMAGE_URI):latest

CONTAINER_ENGINE=$(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)


.PHONY: containerbuild
containerbuild: isclean
	$(CONTAINER_ENGINE) build build --tag $(IMAGE_URI_VERSION)
	$(CONTAINER_ENGINE) tag $(IMAGE_URI_VERSION) $(IMAGE_URI_LATEST)

.PHONY: push
push:
	$(CONTAINER_ENGINE) push $(IMAGE_URI_VERSION)
	$(CONTAINER_ENGINE) push $(IMAGE_URI_LATEST)

.PHONY: skopeo-push
skopeo-push: containerbuild
	skopeo copy \
		--dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
		"docker-daemon:${IMAGE_URI_VERSION}" \
		"docker://${IMAGE_URI_VERSION}"
	skopeo copy \
		--dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
		"docker-daemon:${IMAGE_URI_LATEST}" \
		"docker://${IMAGE_URI_LATEST}"