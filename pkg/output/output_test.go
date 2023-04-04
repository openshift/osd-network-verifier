package output

import (
	"errors"
	"testing"

	nverr "github.com/openshift/osd-network-verifier/pkg/errors"
)

func TestGetEgressURLFailures(t *testing.T) {
	tests := []struct {
		name     string
		o        *Output
		expected int
	}{
		{
			name:     "No egress failures",
			o:        &Output{},
			expected: 0,
		},
		{
			name: "Only egress failures",
			o: &Output{
				failures: []error{
					nverr.NewEgressURLError("www.example.com:443"),
					nverr.NewEgressURLError("www.example.com:80"),
				},
			},
			expected: 2,
		},
		{
			name: "Mixture of failures",
			o: &Output{
				failures: []error{
					nverr.NewGenericError(errors.New("oops")),
					nverr.NewEgressURLError("www.example.com:443"),
					errors.New("idk"),
				},
			},
			expected: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failures := test.o.GetEgressURLFailures()
			if test.expected != len(failures) {
				t.Errorf("expected %d failures, got %d: %v", test.expected, len(failures), failures)
			}
		})
	}
}
