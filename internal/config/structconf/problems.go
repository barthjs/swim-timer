package structconf

import (
	"errors"
	"fmt"
)

// Problems collects validation errors, so validating a configuration reports
// every invalid setting at once. Its zero value is ready to use.
type Problems struct {
	errs []error
}

// Add records err unless it is nil. A joined error is unpacked, so each setting stays on its own line.
func (p *Problems) Add(err error) {
	if err == nil {
		return
	}

	// Only a joined error itself is unpacked: a wrapped one keeps its context and stays one problem
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, inner := range joined.Unwrap() {
			p.Add(inner)
		}
		return
	}

	p.errs = append(p.errs, err)
}

// Addf records an error built from format and args like fmt.Errorf.
func (p *Problems) Addf(format string, args ...any) {
	p.Add(fmt.Errorf(format, args...))
}

// Err returns the recorded errors joined, one per line, or nil if there are none.
func (p *Problems) Err() error {
	return errors.Join(p.errs...)
}
