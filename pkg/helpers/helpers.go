package helpers

import (
	_ "embed"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/errors"
)

//go:embed config/userdata.yaml
var UserdataTemplate string

// PollImmediate calls the condition function at the specified interval up to the specified timeout
// until the condition function returns true or an error
func PollImmediate(interval time.Duration, timeout time.Duration, condition func() (bool, error)) error {

	var totalTime time.Duration = 0

	for totalTime < timeout {
		cond, err := condition()
		if err != nil {
			return err
		}

		if cond {
			return nil
		}

		time.Sleep(interval)
		totalTime += interval
	}
	return errors.ErrWaitTimeout
}
