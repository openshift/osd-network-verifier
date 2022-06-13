package errors

import (
	"errors"
	"fmt"
)

var ErrWaitTimeout = errors.New("timed out waiting for the condition")

type EgressURLError struct {
	e string
}

func (e *EgressURLError) Error() string { return e.e }
func NewEgressURLError(failure string) error {
	return &EgressURLError{
		e: fmt.Sprintf("egressURL error: Unable to reach %s", failure),
	}
}

type UnhandledError struct {
	e string
}

func (e *UnhandledError) Error() string          { return e.e }
func (e *UnhandledError) ErrWaitTimeout() string { return e.e }
func NewGenericUnhandledError(err error) error {
	return &UnhandledError{
		e: fmt.Sprintf("generic unhandled error: %s ", err.Error()),
	}
}

type GenericError struct {
	message string
}

func (e *GenericError) Error() string { return e.message }
func NewGenericError(message string) error {
	return &GenericError{
		message: fmt.Sprintf("network verifier error: %s", message),
	}
}
