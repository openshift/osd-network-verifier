package output

import (
	"fmt"
)

// Output can be used when showcasing validation results at the end of the execution.
// `failures` represents the failed validation tests
// `exceptions` is to show edge cases where onv couldn't be ended up as expected
// `errors` is collection of unhandled errors
type Output struct {
	failures   []string
	exceptions []string
	errors     []error
}

func (o *Output) AddError(err error) {
	if err != nil {
		o.errors = append(o.errors, err)
	}
}

func (o *Output) AddErrorAndReturn(err error) *Output {
	o.AddError(err)
	return o
}

func (o *Output) AddException(message string) {
	o.exceptions = append(o.exceptions, message)
}

func (o *Output) SetFailures(failures []string) {
	o.failures = failures
}

func (o *Output) IsSuccessful() bool {
	if len(o.errors) > 0 || len(o.exceptions) > 0 || len(o.failures) > 0 {
		return false
	}

	return true
}

func (o *Output) printFailures() {
	fmt.Println("printing out failures:")
	for _, v := range o.failures {
		fmt.Println(" - ", v)
	}
}

func (o *Output) printExceptions() {
	fmt.Println("printing out exceptions preventing onv from running:")
	for _, v := range o.exceptions {
		fmt.Println(" - ", v)
	}
}

func (o *Output) printErrors() {
	fmt.Println("printing out errors faced during the execution:")
	for _, v := range o.errors {
		fmt.Println(" - ", v.Error())
	}
}

func (o *Output) Summary() {
	fmt.Println("Summary:")
	o.printFailures()
	o.printExceptions()
	o.printErrors()
}

func (o *Output) Parse() ([]string, []string, []error) {
	return o.failures, o.exceptions, o.errors
}
