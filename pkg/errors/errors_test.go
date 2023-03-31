package errors

import (
	"errors"
	"testing"
)

func TestNewEgressURLError(t *testing.T) {
	tests := []struct {
		url string
	}{
		{
			url: "www.example.com:443",
		},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			err := NewEgressURLError(test.url)
			var nve *GenericError
			if errors.As(err, &nve) {
				if nve.egressURL != test.url {
					t.Errorf("expected %v, got %v", test.url, nve.egressURL)
				}
			}
		})
	}
}
