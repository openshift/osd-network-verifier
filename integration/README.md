# osd-network-verifier integration test

This houses an integration test that will create a vanilla BYOVPC, run osd-network-verifier against it, and then
tears down the BYOVPC.

```bash
DEV_ACCOUNT_ID=""
REGION=""
export $(osdctl account cli -i ${DEV_ACCOUNT_ID}-p osd-staging-2 -r ${REGION} -oenv | xargs)
go build .
./integration --region "${REGION}"
```

## Maintenance

If this hasn't been run a while, you will probably want to bump the dependency on ONV via

```bash
go get -u github.com/openshift/osd-network-verifier
```