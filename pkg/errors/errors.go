package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/smithy-go"
)

type EgressURLError struct {
	e string
}

var ErrWaitTimeout = errors.New("timed out waiting for the condition")

func (e *GenericError) ErrWaitTimeout() string { return e.message }

func (e *EgressURLError) Error() string { return e.e }

func NewEgressURLError(message string) error {
	return &EgressURLError{
		e: fmt.Sprintf("egressURL error: %s", message),
	}
}

type GenericError struct {
	message string
}

func (e *GenericError) Error() string { return e.message }

func NewGenericError(err error) *GenericError {
	var (
		oe *smithy.OperationError
		ae smithy.APIError
	)

	// Generically aws-sdk-go-v2 errors
	if errors.As(err, &oe) {
		if errors.As(oe.Unwrap(), &ae) {
			switch {
			case ae.ErrorCode() == "UnauthorizedOperation":
				if oe.Service() == "EC2" && oe.Operation() == "RunInstances" {
					// AWS will return an UnauthorizedOperation for ec2:RunInstances even if the true error is that
					// it cannot add tags to the instance via ec2:CreateTags
					return &GenericError{
						message: fmt.Sprintf("missing required permission(s) %s:%s and/or ec2:CreateTags", strings.ToLower(oe.Service()), oe.Operation()),
					}
				}
				return &GenericError{
					message: fmt.Sprintf("missing required permission %s:%s", strings.ToLower(oe.Service()), oe.Operation()),
				}
			default:
				return &GenericError{
					message: fmt.Sprintf("error performing %s:%s: %s", strings.ToLower(oe.Service()), oe.Operation(), ae.ErrorMessage()),
				}
			}
		}
	}

	// Just feed forward other generic errors
	return &GenericError{
		message: fmt.Sprintf("network verifier error: %s", err.Error()),
	}
}

type UnhandledError struct {
	message string
}

func (e *UnhandledError) Error() string          { return e.message }
func (e *UnhandledError) ErrWaitTimeout() string { return e.message }
func NewGenericUnhandledError(err error) error {
	return &UnhandledError{
		message: fmt.Sprintf("generic unhandled error: %s ", err.Error()),
	}
}
