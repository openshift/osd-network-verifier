package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/smithy-go"
)

type GenericError struct {
	egressURL string
	message   string
}

type KmsError struct {
	message string
}

func (e *GenericError) Error() string {
	return e.message
}

func (e *GenericError) EgressURL() string {
	return e.egressURL
}

func (k *KmsError) Error() string {
	return k.message
}

// Ensure GenericError implements the error interface
var _ error = &GenericError{}
var _ error = &KmsError{}

// NewGenericError does some preprocessing if the provided error contains an aws-sdk-go-v2 error, otherwise just
// prepends `network verifier error: `
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
				return &GenericError{
					message: fmt.Sprintf("missing required permission %s:%s with error: %s", strings.ToLower(oe.Service()), oe.Operation(), oe.Error()),
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
		message: fmt.Sprintf("network verifier error: %s", err),
	}
}

// NewEgressURLError prepends the provided message with `egressURL error: `
func NewEgressURLError(url string) error {
	return &GenericError{
		egressURL: url,
		message:   fmt.Sprintf("egressURL error: %s", url),
	}
}

func NewKmsError(msg string) error {
	return &KmsError{
		message: msg,
	}
}
