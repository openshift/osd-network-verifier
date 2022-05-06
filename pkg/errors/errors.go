package errors

import (
	_ "embed"
	"errors"
	"fmt"
)

type EgressURLError struct {
	e string
}

var ErrWaitTimeout = errors.New("timed out waiting for the condition")

func (e *GenericError) ErrWaitTimeout() string { return e.e }

func (e *EgressURLError) Error() string { return e.e }

func NewEgressURLError(failure string) error {
	return &EgressURLError{
		e: fmt.Sprintf("egressURL error: %s", failure),
	}
}

type GenericError struct {
	e string
}

func (e *GenericError) Error() string { return e.e }

func NewGenericError(err error) error {
	return &GenericError{
		e: fmt.Sprintf("generic(unhandled) error: %s ", err.Error()),
	}
}
