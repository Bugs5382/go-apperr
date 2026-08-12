// Command registry shows a consumer registering its own codes and turning
// internal errors into sanitized, quotable client messages. The service claims
// the leading digit 1 with WithService, so every code it owns starts with 1.
package main

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

	apperr "github.com/Bugs5382/go-apperr"
)

func main() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "database", Cause: "database unavailable"},
		{Code: 1002, Title: "upstream", Cause: "upstream timeout"},
		{Code: 1003, Title: "order", Cause: "order not found"},
	}, apperr.WithService(1))
	if err != nil {
		panic(err)
	}

	// A coded internal error: the client sees only the sanitized message + code.
	internal := apperr.Coded(1002, errors.New("dial upstream: i/o timeout"))
	msg, code := reg.Present(internal, 1001)
	fmt.Printf("client: %q (code %d)\n", msg, code)

	// An uncoded error falls back to the default code.
	msg, code = reg.Present(errors.New("something unexpected"), 1001)
	fmt.Printf("fallback: %q (code %d)\n", msg, code)

	// The operator can look up what a code means.
	if e, ok := reg.Describe(1002); ok {
		fmt.Printf("operator: %d %s -- %s\n", e.Code, e.Title, e.Cause)
	}

	fmt.Printf("message for any code: %q\n", reg.Message(1099))
}
