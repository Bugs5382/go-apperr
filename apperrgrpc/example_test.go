package apperrgrpc_test

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
	"context"
	"errors"
	"fmt"

	apperr "github.com/Bugs5382/go-apperr"
	"github.com/Bugs5382/go-apperr/apperrgrpc"
	"google.golang.org/grpc/status"
)

// ExampleError returns a coded error from a gRPC handler, then reads it back
// on the client side.
func ExampleError() {
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{
			Code: 6010, Title: "swap", Cause: "candidate not eligible",
			Symbol: "SWAP_NOT_ELIGIBLE", Category: apperr.CategoryFailedPrecondition,
			UserSafe: true, Message: "That person isn't an eligible approver.",
		},
	})
	if err != nil {
		panic(err)
	}

	// Server side.
	refused := apperr.WithMeta(apperr.Coded(6010, errors.New("not in pool")), apperr.Meta("new_user_id", "u-2"))
	sent := apperrgrpc.Error(context.Background(), reg, refused, 6000, "workflow.example.org")

	// Client side.
	st := status.Convert(sent)
	info, _ := apperrgrpc.FromError(sent)
	fmt.Println(st.Code(), "|", st.Message())
	fmt.Println(info.Code, info.Symbol, info.Domain, info.Metadata["new_user_id"])
	// Output:
	// FailedPrecondition | That person isn't an eligible approver.
	// 6010 SWAP_NOT_ELIGIBLE workflow.example.org u-2
}

// ExampleCode maps a category to its gRPC code.
func ExampleCode() {
	fmt.Println(apperrgrpc.Code(apperr.CategoryUnauthenticated))
	fmt.Println(apperrgrpc.Code(apperr.CategoryInternal))
	// Output:
	// Unauthenticated
	// Internal
}
