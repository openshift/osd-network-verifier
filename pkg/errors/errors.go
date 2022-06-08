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
		e: fmt.Sprintf("egressURL error: %s", failure),
	}
}

type UnhandledError struct {
	e string
}

func (e *UnhandledError) Error() string          { return e.e }
func (e *UnhandledError) ErrWaitTimeout() string { return e.e }
func NewGenericError(err error) error {
	return &UnhandledError{
		e: fmt.Sprintf("Unhandled error: %s ", err.Error()),
	}
}

type GenericNetworkVerifierError struct {
	e string
}

func (e *GenericNetworkVerifierError) Error() string { return e.e }
func NewGenericNetworkVerifierError(failure string) error {
	return &EgressURLError{
		e: fmt.Sprintf("Generic Network Verifier error: %s", failure),
	}
}