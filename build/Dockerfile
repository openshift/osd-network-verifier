FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV NETWORK_VALIDATOR_BIN=network-validator

WORKDIR /root/
COPY /config.yaml /root/
COPY /network-validator /usr/local/bin/

ENTRYPOINT /usr/local/bin/network-validator --config=/root/config.yaml --timeout=1s