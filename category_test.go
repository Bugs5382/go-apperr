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

func newCategoryRegistry(t *testing.T) *apperr.Registry {
	t.Helper()
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "database", Cause: "database unavailable"},
		{Code: 1002, Title: "widget", Cause: "widget not found", Category: apperr.CategoryNotFound},
		{Code: 1003, Title: "input", Cause: "bad widget id", Category: apperr.CategoryInvalid},
		{Code: 1004, Title: "upstream", Cause: "upstream down", Category: apperr.CategoryUnavailable},
		{Code: 1005, Title: "auth", Cause: "caller lacks role", Category: apperr.CategoryPermissionDenied},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func TestCategoryDefaultsToInternal(t *testing.T) {
	t.Parallel()
	var e apperr.Entry
	if e.Category != apperr.CategoryInternal {
		t.Fatalf("zero Entry category = %v; want internal", e.Category)
	}
	reg := newCategoryRegistry(t)
	if got := reg.Category(apperr.Coded(1001, errors.New("x"))); got != apperr.CategoryInternal {
		t.Fatalf("Category(1001) = %v; want internal", got)
	}
}

func TestCategoryFromCodedError(t *testing.T) {
	t.Parallel()
	reg := newCategoryRegistry(t)

	tests := []struct {
		code int
		want apperr.Category
	}{
		{1002, apperr.CategoryNotFound},
		{1003, apperr.CategoryInvalid},
		{1004, apperr.CategoryUnavailable},
		{1005, apperr.CategoryPermissionDenied},
	}
	for _, tc := range tests {
		// Wrap the coded error further so the lookup has to walk the chain.
		err := fmt.Errorf("handler: %w", apperr.Coded(tc.code, errors.New("cause")))
		if got := reg.Category(err); got != tc.want {
			t.Errorf("Category(%d) = %v; want %v", tc.code, got, tc.want)
		}
	}
}

func TestCategoryFallsBackToInternal(t *testing.T) {
	t.Parallel()
	reg := newCategoryRegistry(t)

	if got := reg.Category(errors.New("uncoded")); got != apperr.CategoryInternal {
		t.Fatalf("uncoded error category = %v; want internal", got)
	}
	if got := reg.Category(apperr.Coded(9999, nil)); got != apperr.CategoryInternal {
		t.Fatalf("unregistered code category = %v; want internal", got)
	}
	if got := reg.Category(nil); got != apperr.CategoryInternal {
		t.Fatalf("nil error category = %v; want internal", got)
	}
}

func TestDescribeCarriesCategory(t *testing.T) {
	t.Parallel()
	reg := newCategoryRegistry(t)
	e, ok := reg.Describe(1002)
	if !ok || e.Category != apperr.CategoryNotFound {
		t.Fatalf("Describe(1002) = %+v,%t; want not-found", e, ok)
	}
}

func TestCategoryString(t *testing.T) {
	t.Parallel()
	tests := map[apperr.Category]string{
		apperr.CategoryInternal:         "internal",
		apperr.CategoryNotFound:         "not-found",
		apperr.CategoryInvalid:          "invalid",
		apperr.CategoryUnavailable:      "unavailable",
		apperr.CategoryPermissionDenied: "permission-denied",
		apperr.Category(99):             "Category(99)",
	}
	for c, want := range tests {
		if got := c.String(); got != want {
			t.Errorf("Category(%d).String() = %q; want %q", int(c), got, want)
		}
	}
}

func TestNewRegistryRejectsUnknownCategory(t *testing.T) {
	t.Parallel()
	_, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, Category: apperr.Category(99)}})
	if err == nil {
		t.Fatal("expected an error for an unknown category")
	}
}

func TestMarkdownUnchangedByCategory(t *testing.T) {
	t.Parallel()
	// The docs table stays Code | Area | Cause, so existing drift tests keep
	// passing after a consumer adds categories.
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1002, Title: "widget", Cause: "widget not found", Category: apperr.CategoryNotFound},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	want := "| Code | Area | Cause |\n| --- | --- | --- |\n| 1002 | widget | widget not found |\n"
	if got := reg.Markdown(); got != want {
		t.Fatalf("Markdown() =\n%q\nwant\n%q", got, want)
	}
}

// grpcCode maps a category to a gRPC status code number. The numbers are the
// stable values from the gRPC spec (NotFound is 5, and so on), written out so
// this package never imports a transport. A real mapper would return
// codes.Code from google.golang.org/grpc/codes instead of an int.
func grpcCode(c apperr.Category) int {
	switch c {
	case apperr.CategoryNotFound:
		return 5 // codes.NotFound
	case apperr.CategoryInvalid:
		return 3 // codes.InvalidArgument
	case apperr.CategoryUnavailable:
		return 14 // codes.Unavailable
	case apperr.CategoryPermissionDenied:
		return 7 // codes.PermissionDenied
	default:
		return 13 // codes.Internal
	}
}

// httpStatus maps a category to an HTTP status. A real mapper would use the
// net/http constants (http.StatusNotFound, and so on).
func httpStatus(c apperr.Category) int {
	switch c {
	case apperr.CategoryNotFound:
		return 404
	case apperr.CategoryInvalid:
		return 400
	case apperr.CategoryUnavailable:
		return 503
	case apperr.CategoryPermissionDenied:
		return 403
	default:
		return 500
	}
}

// ExampleRegistry_Category_grpc shows one shared gRPC mapper: the handler
// looks up the category from the coded error and turns it into a status code,
// with the sanitized message from Present as the status message.
func ExampleRegistry_Category_grpc() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1002, Title: "widget", Cause: "widget not found", Category: apperr.CategoryNotFound},
	})
	if err != nil {
		panic(err)
	}
	internal := apperr.Coded(1002, errors.New("select widget: no rows"))
	msg, _ := reg.Present(internal, 1000)
	// In a real server: status.Error(codes.Code(grpcCode(...)), msg)
	fmt.Printf("grpc code %d: %s\n", grpcCode(reg.Category(internal)), msg)
	// Output: grpc code 5: Code 1002: Internal Error
}

// ExampleRegistry_Category_http shows the same idea for HTTP: one mapper from
// category to status, shared by every handler.
func ExampleRegistry_Category_http() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1005, Title: "auth", Cause: "caller lacks role", Category: apperr.CategoryPermissionDenied},
	})
	if err != nil {
		panic(err)
	}
	internal := apperr.Coded(1005, errors.New("role check failed"))
	msg, _ := reg.Present(internal, 1000)
	// In a real handler: http.Error(w, msg, httpStatus(...))
	fmt.Printf("http %d: %s\n", httpStatus(reg.Category(internal)), msg)
	// Output: http 403: Code 1005: Internal Error
}
