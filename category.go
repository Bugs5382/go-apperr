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

import "strconv"

// Category is a transport-neutral class of failure for a registered code. It
// lets one shared mapper per transport (a gRPC status, an HTTP status) turn any
// coded error into the right response, instead of every service keeping its
// own code-to-status table. The package never imports a transport; see the
// Registry.Category examples for gRPC and HTTP mappers.
//
// The zero value is CategoryInternal, so an Entry that sets no Category is
// internal, exactly as before categories existed.
type Category int

const (
	// CategoryInternal is an unexpected server-side failure. It is the default.
	CategoryInternal Category = iota
	// CategoryNotFound means the requested resource does not exist.
	CategoryNotFound
	// CategoryInvalid means the caller sent a malformed or invalid request.
	CategoryInvalid
	// CategoryUnavailable means a dependency is down or overloaded and the
	// caller may retry later.
	CategoryUnavailable
	// CategoryPermissionDenied means the caller is known but not allowed to do
	// this.
	CategoryPermissionDenied

	// categoryCount marks the end of the known categories for validation.
	categoryCount
)

// categoryNames holds the stable, kebab-case name of each known category.
var categoryNames = [categoryCount]string{
	CategoryInternal:         "internal",
	CategoryNotFound:         "not-found",
	CategoryInvalid:          "invalid",
	CategoryUnavailable:      "unavailable",
	CategoryPermissionDenied: "permission-denied",
}

// String returns the category's stable name ("internal", "not-found",
// "invalid", "unavailable", "permission-denied"), suitable for a log field or
// span attribute. An unknown value renders as "Category(n)".
func (c Category) String() string {
	if c.valid() {
		return categoryNames[c]
	}
	return "Category(" + strconv.Itoa(int(c)) + ")"
}

// valid reports whether c is one of the declared categories.
func (c Category) valid() bool {
	return c >= 0 && c < categoryCount
}

// Category returns the category registered for the code carried by err. It
// walks the error chain the same way Code does. An uncoded error, a nil error,
// or a code that is not registered is CategoryInternal, so a mapper always has
// a safe answer.
func (r *Registry) Category(err error) Category {
	code, ok := Code(err)
	if !ok {
		return CategoryInternal
	}
	return r.entries[code].Category
}
