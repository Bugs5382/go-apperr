package apperr

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"errors"
	"fmt"
)

// codedError wraps a cause with a stable numeric code. It is the value Coded
// returns; the code is unexported so callers read it through Code rather than a
// type assertion, keeping the surface small.
type codedError struct {
	code  int
	cause error
}

// Coded wraps cause with a stable numeric code. The returned error's Unwrap
// returns cause, so errors.Is and errors.As traverse to it unchanged; the code
// rides alongside and is recovered with Code. A nil cause is allowed -- the
// result is a bare coded marker whose Error reports only the code.
func Coded(code int, cause error) error {
	return &codedError{code: code, cause: cause}
}

// Error renders the code and the wrapped cause. It is intended for logs and
// internal diagnostics, not for clients -- use Registry.Present for the
// sanitized, client-facing message.
func (e *codedError) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("code %d", e.code)
	}
	return fmt.Sprintf("code %d: %v", e.code, e.cause)
}

// Unwrap returns the wrapped cause so errors.Is and errors.As traverse the
// chain past this coded error.
func (e *codedError) Unwrap() error { return e.cause }

// Code returns the code carried by err, walking the error chain with errors.As
// and returning the nearest coded error's code. It reports (0, false) when err
// is nil or carries no code, so a caller can distinguish "no code" from a
// legitimately zero code.
func Code(err error) (code int, ok bool) {
	if err == nil {
		return 0, false
	}
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code, true
	}
	return 0, false
}
