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
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

// fieldSink captures the request fields each sink sees, using only the
// existing Recorder and Logger signatures.
type fieldSink struct {
	recorded []apperr.Field
	logged   []apperr.Field
	calls    int
}

func (s *fieldSink) RecordCode(ctx context.Context, _ int, _ error) {
	s.calls++
	s.recorded = apperr.FieldsFromContext(ctx)
}

func (s *fieldSink) LogCoded(ctx context.Context, _ int, _ error) {
	s.calls++
	s.logged = apperr.FieldsFromContext(ctx)
}

func TestFieldsReachBothSinks(t *testing.T) {
	t.Parallel()
	sink := &fieldSink{}
	reg := newTestRegistry(t, apperr.WithRecorder(sink), apperr.WithLogger(sink))

	ctx := apperr.ContextWithFields(context.Background(),
		apperr.Field{Key: "method", Value: "GET"},
		apperr.Field{Key: "route", Value: "/widgets/{id}"},
	)
	if _, code := reg.PresentContext(ctx, apperr.Coded(1002, errors.New("boom")), 1001); code != 1002 {
		t.Fatalf("code = %d; want 1002", code)
	}

	want := []apperr.Field{{Key: "method", Value: "GET"}, {Key: "route", Value: "/widgets/{id}"}}
	if sink.calls != 2 {
		t.Fatalf("sink calls = %d; want 2", sink.calls)
	}
	if !reflect.DeepEqual(sink.recorded, want) {
		t.Fatalf("recorder fields = %v; want %v", sink.recorded, want)
	}
	if !reflect.DeepEqual(sink.logged, want) {
		t.Fatalf("logger fields = %v; want %v", sink.logged, want)
	}
}

func TestNoFieldsReachSinksAsNil(t *testing.T) {
	t.Parallel()
	sink := &fieldSink{}
	reg := newTestRegistry(t, apperr.WithRecorder(sink), apperr.WithLogger(sink))

	reg.PresentContext(context.Background(), apperr.Coded(1002, errors.New("boom")), 1001)
	if sink.calls != 2 {
		t.Fatalf("sink calls = %d; want 2", sink.calls)
	}
	if sink.recorded != nil || sink.logged != nil {
		t.Fatalf("fields = %v / %v; want nil for a context with none", sink.recorded, sink.logged)
	}
}

func TestContextWithFieldsAppendsWithoutTouchingParent(t *testing.T) {
	t.Parallel()

	parent := apperr.ContextWithFields(context.Background(), apperr.Field{Key: "method", Value: "POST"})
	childA := apperr.ContextWithFields(parent, apperr.Field{Key: "route", Value: "/a"})
	childB := apperr.ContextWithFields(parent, apperr.Field{Key: "route", Value: "/b"})

	if got := apperr.FieldsFromContext(parent); len(got) != 1 {
		t.Fatalf("parent fields = %v; want one field", got)
	}
	wantA := []apperr.Field{{Key: "method", Value: "POST"}, {Key: "route", Value: "/a"}}
	if got := apperr.FieldsFromContext(childA); !reflect.DeepEqual(got, wantA) {
		t.Fatalf("childA fields = %v; want %v", got, wantA)
	}
	wantB := []apperr.Field{{Key: "method", Value: "POST"}, {Key: "route", Value: "/b"}}
	if got := apperr.FieldsFromContext(childB); !reflect.DeepEqual(got, wantB) {
		t.Fatalf("childB fields = %v; want %v", got, wantB)
	}
}

func TestFieldsFromContextReturnsCopy(t *testing.T) {
	t.Parallel()

	ctx := apperr.ContextWithFields(context.Background(), apperr.Field{Key: "method", Value: "GET"})
	got := apperr.FieldsFromContext(ctx)
	got[0].Value = "changed"
	if again := apperr.FieldsFromContext(ctx); again[0].Value != "GET" {
		t.Fatalf("stored field changed to %v; a caller must not be able to mutate it", again[0].Value)
	}
}

// ExampleContextWithFields attaches request metadata at the edge and reads it
// back the way a sink would, from the ctx it already receives.
func ExampleContextWithFields() {
	ctx := apperr.ContextWithFields(context.Background(),
		apperr.Field{Key: "method", Value: "GET"},
		apperr.Field{Key: "route", Value: "/widgets/{id}"},
	)
	for _, f := range apperr.FieldsFromContext(ctx) {
		fmt.Printf("%s=%v\n", f.Key, f.Value)
	}
	// Output:
	// method=GET
	// route=/widgets/{id}
}

func TestContextWithNoFieldsIsUnchanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if got := apperr.ContextWithFields(ctx); got != ctx {
		t.Fatal("ContextWithFields with no fields should return ctx unchanged")
	}
	//nolint:staticcheck // a nil context must not panic in a sink.
	if got := apperr.FieldsFromContext(nil); got != nil {
		t.Fatalf("FieldsFromContext(nil) = %v; want nil", got)
	}
}
