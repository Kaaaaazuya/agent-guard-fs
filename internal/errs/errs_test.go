package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOf(t *testing.T) {
	denied := New(PolicyDenied, "denied by rule")
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{"direct", denied, PolicyDenied},
		{"wrapped", fmt.Errorf("guard: %w", denied), PolicyDenied},
		{"plain error", errors.New("boom"), Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeOf(tt.err); got != tt.want {
				t.Errorf("CodeOf() = %q, want %q", got, tt.want)
			}
		})
	}
}
