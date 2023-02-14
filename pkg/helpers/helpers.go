package helpers

import (
	_ "embed"
	"errors"
	"math/rand"
	"time"
)

//go:embed config/userdata.yaml
var UserdataTemplate string

// RandSeq generates random string with n characters.
func RandSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

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

	return errors.New("timed out waiting for the condition")
}

// Enumerated type representing the platform underlying the cluster-under-test
const (
	PLATFORM_AWS           string = "aws"
	PLATFORM_GCP           string = "gcp"
	PLATFORM_HOSTEDCLUSTER string = "hostedcluster"
)
