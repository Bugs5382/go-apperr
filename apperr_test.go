package apperr_test

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
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

var errDatabaseDown = errors.New("database unavailable")

func TestCodeResolvesNearest(t *testing.T) {
	t.Parallel()

	// A code wrapped, then re-coded: Code returns the nearest (outermost) one.
	inner := apperr.Coded(1001, errDatabaseDown)
	outer := apperr.Coded(1002, fmt.Errorf("handle request: %w", inner))

	code, ok := apperr.Code(outer)
	if !ok || code != 1002 {
		t.Fatalf("Code(outer) = %d,%t; want 1002,true", code, ok)
	}

	code, ok = apperr.Code(inner)
	if !ok || code != 1001 {
		t.Fatalf("Code(inner) = %d,%t; want 1001,true", code, ok)
	}
}

func TestCodeAbsentAndNil(t *testing.T) {
	t.Parallel()

	if code, ok := apperr.Code(nil); ok || code != 0 {
		t.Fatalf("Code(nil) = %d,%t; want 0,false", code, ok)
	}
	if code, ok := apperr.Code(errors.New("plain")); ok || code != 0 {
		t.Fatalf("Code(plain) = %d,%t; want 0,false", code, ok)
	}
}

func TestUnwrapTraversal(t *testing.T) {
	t.Parallel()

	err := apperr.Coded(1001, fmt.Errorf("load widget: %w", errDatabaseDown))

	if !errors.Is(err, errDatabaseDown) {
		t.Fatal("errors.Is should traverse the coded error to the cause")
	}
	if got := errors.Unwrap(err); got == nil {
		t.Fatal("Unwrap should return the wrapped cause")
	}
}

func TestCodedNilCause(t *testing.T) {
	t.Parallel()

	err := apperr.Coded(1001, nil)
	if code, ok := apperr.Code(err); !ok || code != 1001 {
		t.Fatalf("Code = %d,%t; want 1001,true", code, ok)
	}
	if err.Error() == "" {
		t.Fatal("Error() should be non-empty for a nil cause")
	}
}
