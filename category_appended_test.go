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
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

func TestNewCategoriesAppended(t *testing.T) {
	t.Parallel()
	tests := []struct {
		c     apperr.Category
		value int
		name  string
	}{
		{apperr.CategoryInternal, 0, "internal"},
		{apperr.CategoryNotFound, 1, "not-found"},
		{apperr.CategoryInvalid, 2, "invalid"},
		{apperr.CategoryUnavailable, 3, "unavailable"},
		{apperr.CategoryPermissionDenied, 4, "permission-denied"},
		{apperr.CategoryFailedPrecondition, 5, "failed-precondition"},
		{apperr.CategoryDeadlineExceeded, 6, "deadline-exceeded"},
		{apperr.CategoryUnauthenticated, 7, "unauthenticated"},
		{apperr.CategoryAlreadyExists, 8, "already-exists"},
		{apperr.Category(9), 9, "Category(9)"},
	}
	for _, tc := range tests {
		if int(tc.c) != tc.value {
			t.Errorf("%s = %d; want %d", tc.name, int(tc.c), tc.value)
		}
		if got := tc.c.String(); got != tc.name {
			t.Errorf("Category(%d).String() = %q; want %q", tc.value, got, tc.name)
		}
	}
}

func TestNewCategoriesRegisterAndResolve(t *testing.T) {
	t.Parallel()
	cats := []apperr.Category{
		apperr.CategoryFailedPrecondition,
		apperr.CategoryDeadlineExceeded,
		apperr.CategoryUnauthenticated,
		apperr.CategoryAlreadyExists,
	}
	entries := make([]apperr.Entry, len(cats))
	for i, c := range cats {
		entries[i] = apperr.Entry{Code: 1001 + i, Category: c}
	}
	reg, err := apperr.NewRegistry(entries)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	for i, c := range cats {
		if got := reg.Category(apperr.Coded(1001+i, errors.New("x"))); got != c {
			t.Errorf("Category(%d) = %v; want %v", 1001+i, got, c)
		}
	}
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1, Category: apperr.Category(9)}}); err == nil {
		t.Fatal("Category(9) should still be unknown")
	}
}
