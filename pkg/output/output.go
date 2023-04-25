package output

import (
	"errors"
	"fmt"

	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

const logFormat = " - %v\n"

// Output can be used when showcasing validation results at the end of the execution.
type Output struct {
	// debugLogs
	debugLogs []string
	// failures represents the failed validation tests
	failures []error
	// exceptions is to show edge cases where a verifier test couldn't be ran as expected
	exceptions []error
	// errors is collection of unhandled errors
	errors []error
}

func (o *Output) AddDebugLogs(log string) {
	o.debugLogs = append(o.debugLogs, log)
}

// AddError adds error as generic to the list of errors
func (o *Output) AddError(err error) *Output {
	if err != nil {
		o.errors = append(o.errors, handledErrors.NewGenericError(err))
	}

	return o
}

// AddException adds an exception to the list of exceptions
func (o *Output) AddException(message error) {
	o.exceptions = append(o.exceptions, message)
}

// SetEgressFailures sets egress endpoint failures as a bulk update
func (o *Output) SetEgressFailures(failures []string) {
	for _, f := range failures {
		o.failures = append(o.failures, handledErrors.NewEgressURLError(f))
	}
}

// IsSuccessful checks whether the output contains any item, returns false if there's any
func (o *Output) IsSuccessful() bool {
	if len(o.errors) > 0 || len(o.exceptions) > 0 || len(o.failures) > 0 {
		return false
	}

	return true
}

// Format can be used to retrieve the string for the output structure
func (o *Output) Format(debug bool) string {
	if o == nil {
		return ""
	}
	output := ""
	if debug {
		output += "printing out debug logs from the execution:\n"
		output += format(o.debugLogs)
	}
	if o.IsSuccessful() {
		output += "All tests passed!\n"
		return output
	}
	output += "printing out failures:\n"
	output += format(o.failures)
	output += "printing out exceptions preventing the verifier from running the specific test:\n"
	output += format(o.exceptions)
	output += "printing out errors faced during the execution:\n"
	output += format(o.errors)
	return output
}

func format[T any](slice []T) string {
	if len(slice) == 0 {
		return ""
	}
	output := ""
	for _, value := range slice {
		output += fmt.Sprintf(logFormat, value)
	}
	return output + "\n"
}

// Summary can be used for printing out output structure
func (o *Output) Summary(debug bool) {
	fmt.Println("Summary:")
	fmt.Print(o.Format(debug))
}

// Parse returns the data being stored on output
// - failures as []error
// - exceptions as []error
// - errors as []error
func (o *Output) Parse() ([]error, []error, []error) {
	return o.failures, o.exceptions, o.errors
}

// GetEgressURLFailures returns only errors related to network egress failures.
// Use the EgressURL() method to obtain the specific url for each error.
func (o *Output) GetEgressURLFailures() []*handledErrors.GenericError {
	egressErrs := []*handledErrors.GenericError{}

	for _, err := range o.failures {
		var nve *handledErrors.GenericError
		if errors.As(err, &nve) {
			if nve.EgressURL() != "" {
				egressErrs = append(egressErrs, nve)
			}
		}
	}

	return egressErrs
}
