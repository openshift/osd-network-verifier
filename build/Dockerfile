FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS golang
WORKDIR /golang
RUN microdnf install -y  \
curl \
make \
gcc \
git \
tar \
gcc-c++

RUN curl -fsSLo /tmp/go.tgz https://golang.org/dl/go1.16.6.linux-amd64.tar.gz; \
mkdir -p bin; \
tar -C /golang -xzf /tmp/go.tgz; \
rm /tmp/go.tgz; \
export PATH="/golang/go/bin:${PATH}"; \
go version

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS builder
WORKDIR /app
COPY --from=golang /golang/go /usr/local
COPY . .
RUN pushd bin; go mod init validator; go get ./...; go build network-validator.go; popd;

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /app
COPY --from=builder /app/bin/network-validator /usr/bin/network-validator
COPY --from=builder /app/config/ /app/config/

ENTRYPOINT ["network-validator", "--config=/app/config/config.yaml"]
