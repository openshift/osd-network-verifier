package errors

import (
	"errors"
	"fmt"
)

type EgressURLError struct {
	e string
}

var ErrWaitTimeout = errors.New("timed out waiting for the condition")

func (e *GenericError) ErrWaitTimeout() string { return e.message }

func (e *EgressURLError) Error() string { return e.e }

func NewEgressURLError(failure string) error {
	return &EgressURLError{
		e: fmt.Sprintf("egressURL error: %s", failure),
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
