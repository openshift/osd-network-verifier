FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS golang
RUN microdnf install -y  \
git  \
go-toolset

FROM golang as builder
WORKDIR /app
COPY build/bin .
RUN go mod download && go build -ldflags="-s -w" .

FROM registry.access.redhat.com/ubi8/ubi-micro:latest
WORKDIR /app
COPY --from=builder /app/network-validator /usr/bin/network-validator
COPY build/config /app/build/config/
# Copy the CA trust bundle from ubi-minimal --> ubi-micro
COPY --from=builder /etc/pki/ca-trust /etc/pki/ca-trust

ENTRYPOINT ["network-validator", "--config=/app/build/config/config.yaml"]
