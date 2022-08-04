package helpers

import (
	_ "embed"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/errors"
)

//go:embed config/userdata.yaml
var UserdataTemplate string

func PollImmediate(interval time.Duration, timeout time.Duration, condition func() (bool, error)) error {

	var totalTime time.Duration = 0

	for totalTime < timeout {
		cond, err := condition()
		if cond {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(interval)
		totalTime += interval
	}
	return errors.ErrWaitTimeout
}
