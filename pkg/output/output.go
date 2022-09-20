package output

import (
	"fmt"

	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

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

func (o *Output) printFailures() {
	if o != nil && len(o.failures) > 0 {
		fmt.Println("printing out failures:")
		for _, v := range o.failures {
			fmt.Println(" - ", v)
		}
	}
}

func (o *Output) printExceptions() {
	if o != nil && len(o.exceptions) > 0 {
		fmt.Println("printing out exceptions preventing the verifier from running the specific test:")
		for _, v := range o.exceptions {
			fmt.Println(" - ", v)
		}
	}
}

func (o *Output) printErrors() {
	if o != nil && len(o.errors) > 0 {
		fmt.Println("printing out errors faced during the execution:")
		for _, v := range o.errors {
			fmt.Println(" - ", v.Error())
		}
	}
}

func (o *Output) printDebugLogs() {
	if o != nil && len(o.debugLogs) > 0 {
		fmt.Println("printing out debug logs from the execution:")
		for _, v := range o.debugLogs {
			fmt.Println(" - ", v)
		}
	}
}

// Summary can be used for printing out output structure
func (o *Output) Summary(debug bool) {
	fmt.Println("Summary:")
	if debug {
		o.printDebugLogs()
	}

	if o.IsSuccessful() {
		fmt.Println("All tests pass!")
	} else {
		o.printFailures()
		o.printExceptions()
		o.printErrors()
	}
}

// Parse returns the data being stored on output
// - failures as []error
// - exceptions as []error
// - errors as []error
func (o *Output) Parse() ([]error, []error, []error) {
	return o.failures, o.exceptions, o.errors
}
