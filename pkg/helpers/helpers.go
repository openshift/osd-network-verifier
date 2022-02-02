package helpers

import (
	"fmt"
	"time"
)

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

	return fmt.Errorf("timed out")
}
