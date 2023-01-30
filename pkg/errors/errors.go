package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/smithy-go"
)

// GenericError implements the error interface
type GenericError struct {
	message string
}

func (e *GenericError) Error() string { return e.message }

// Ensure GenericError implements the error interface
var _ error = &GenericError{}

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
		message: fmt.Sprintf("network verifier error: %s", err.Error()),
	}
}

// NewEgressURLError prepends the provided message with `egressURL error: `
func NewEgressURLError(message string) error {
	return &GenericError{
		message: fmt.Sprintf("egressURL error: %s", message),
	}
}
