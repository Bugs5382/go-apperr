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

	apperr "github.com/Bugs5382/go-apperr"
)

// ExampleWithMeta attaches wire metadata to one coded error: the refused
// candidate's ID travels with the error to the client.
func ExampleWithMeta() {
	err := apperr.WithMeta(
		apperr.Coded(6010, errors.New("candidate not in the eligible pool")),
		apperr.Meta("new_user_id", "u-2"),
	)
	code, _ := apperr.Code(err)
	fmt.Println(code, apperr.Metadata(err)["new_user_id"])
	fmt.Println(err)
	// Output:
	// 6010 u-2
	// code 6010: candidate not in the eligible pool
}

// ExampleEntry_userSafe shows the end user a user-safe entry's message, filled
// from the error's metadata, while an entry that does not opt in keeps the
// generic template.
func ExampleEntry_userSafe() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 6001, Title: "resolver", Cause: "approver lookup failed"},
		{
			Code: 6010, Title: "swap", Cause: "candidate not eligible",
			Symbol: "SWAP_NOT_ELIGIBLE", Category: apperr.CategoryFailedPrecondition,
			UserSafe: true, Message: "That person isn't an eligible approver for stage {stage}.",
		},
	})
	if err != nil {
		panic(err)
	}

	refused := apperr.WithMeta(apperr.Coded(6010, nil), apperr.Meta("stage", "2"))
	msg, _ := reg.Present(refused, 6000)
	fmt.Println(msg)

	msg, _ = reg.Present(apperr.Coded(6001, errors.New("dial tcp: refused")), 6000)
	fmt.Println(msg)
	// Output:
	// That person isn't an eligible approver for stage 2.
	// Code 6001: Internal Error
}

// ExampleCategory lists the categories added after the first five; existing
// constants keep their values.
func ExampleCategory() {
	for _, c := range []apperr.Category{
		apperr.CategoryFailedPrecondition,
		apperr.CategoryDeadlineExceeded,
		apperr.CategoryUnauthenticated,
		apperr.CategoryAlreadyExists,
	} {
		fmt.Println(int(c), c)
	}
	// Output:
	// 5 failed-precondition
	// 6 deadline-exceeded
	// 7 unauthenticated
	// 8 already-exists
}

// ExampleRegistry_Markdown_symbols adds the Symbol and User-safe columns once
// an entry uses them.
func ExampleRegistry_Markdown_symbols() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "database", Cause: "database unavailable"},
		{Code: 1002, Title: "widget", Cause: "widget not found", Symbol: "WIDGET_NOT_FOUND", UserSafe: true, Message: "That widget no longer exists."},
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(reg.Markdown())
	// Output:
	// | Code | Symbol | Area | Cause | User-safe |
	// | --- | --- | --- | --- | --- |
	// | 1001 |  | database | database unavailable | no |
	// | 1002 | `WIDGET_NOT_FOUND` | widget | widget not found | yes |
}
