package helpers

import (
	_ "embed"
	"errors"
	"time"
)

//go:embed config/userdata.yaml
var UserdataTemplate string

var ErrWaitTimeout = errors.New("timed out waiting for the condition")

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

	return ErrWaitTimeout
}
