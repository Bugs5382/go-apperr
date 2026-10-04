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
	"maps"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

func TestMetadataFromWithMeta(t *testing.T) {
	t.Parallel()
	err := apperr.WithMeta(apperr.Coded(6010, errors.New("not eligible")),
		apperr.Meta("new_user_id", "u-2"), apperr.Meta("stage", "1"))
	want := map[string]string{"new_user_id": "u-2", "stage": "1"}
	if got := apperr.Metadata(err); !maps.Equal(got, want) {
		t.Fatalf("Metadata = %v; want %v", got, want)
	}
}

func TestMetadataNoneIsNil(t *testing.T) {
	t.Parallel()
	for _, err := range []error{nil, errors.New("plain"), apperr.Coded(1001, nil)} {
		if got := apperr.Metadata(err); got != nil {
			t.Errorf("Metadata(%v) = %v; want nil", err, got)
		}
	}
}

func TestWithMetaNilErrorStaysNil(t *testing.T) {
	t.Parallel()
	if err := apperr.WithMeta(nil, apperr.Meta("k", "v")); err != nil {
		t.Fatalf("WithMeta(nil) = %v; want nil", err)
	}
}

func TestWithMetaNoPairsReturnsErrUnchanged(t *testing.T) {
	t.Parallel()
	base := apperr.Coded(1001, errors.New("x"))
	if got := apperr.WithMeta(base); got != base {
		t.Fatalf("WithMeta(err) with no pairs = %v; want the same error", got)
	}
}

func TestWithMetaKeepsErrorChain(t *testing.T) {
	t.Parallel()
	cause := errors.New("database unavailable")
	base := apperr.Coded(1001, cause)
	err := apperr.WithMeta(base, apperr.Meta("table", "widgets"))

	if err.Error() != base.Error() {
		t.Fatalf("Error() = %q; want %q", err.Error(), base.Error())
	}
	if code, ok := apperr.Code(err); !ok || code != 1001 {
		t.Fatalf("Code = %d,%t; want 1001,true", code, ok)
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should reach the cause through WithMeta")
	}
}

func TestMetadataOuterLayerWins(t *testing.T) {
	t.Parallel()
	inner := apperr.WithMeta(apperr.Coded(1001, nil), apperr.Meta("k", "inner"), apperr.Meta("a", "1"))
	outer := apperr.WithMeta(fmt.Errorf("handler: %w", inner), apperr.Meta("k", "outer"))
	want := map[string]string{"k": "outer", "a": "1"}
	if got := apperr.Metadata(outer); !maps.Equal(got, want) {
		t.Fatalf("Metadata = %v; want %v", got, want)
	}
}

func TestMetadataLaterPairWinsInOneLayer(t *testing.T) {
	t.Parallel()
	err := apperr.WithMeta(errors.New("x"), apperr.Meta("k", "first"), apperr.Meta("k", "second"))
	if got := apperr.Metadata(err)["k"]; got != "second" {
		t.Fatalf("Metadata[k] = %q; want second", got)
	}
}

func TestMetadataWalksJoinedErrors(t *testing.T) {
	t.Parallel()
	a := apperr.WithMeta(errors.New("a"), apperr.Meta("a", "1"), apperr.Meta("k", "first"))
	b := apperr.WithMeta(errors.New("b"), apperr.Meta("b", "2"), apperr.Meta("k", "second"))
	want := map[string]string{"a": "1", "b": "2", "k": "first"}
	if got := apperr.Metadata(errors.Join(a, b)); !maps.Equal(got, want) {
		t.Fatalf("Metadata = %v; want %v", got, want)
	}
}

func TestMetadataIsACopy(t *testing.T) {
	t.Parallel()
	err := apperr.WithMeta(errors.New("x"), apperr.Meta("k", "v"))
	apperr.Metadata(err)["k"] = "changed"
	if got := apperr.Metadata(err)["k"]; got != "v" {
		t.Fatalf("Metadata[k] = %q after mutating a returned map; want v", got)
	}
}

func TestMetadataSeparateFromFields(t *testing.T) {
	t.Parallel()
	ctx := apperr.ContextWithFields(t.Context(), apperr.Field{Key: "route", Value: "/widgets"})
	err := apperr.WithMeta(errors.New("x"), apperr.Meta("widget_id", "w-1"))
	if _, ok := apperr.Metadata(err)["route"]; ok {
		t.Fatal("request fields must not leak into wire metadata")
	}
	for _, f := range apperr.FieldsFromContext(ctx) {
		if f.Key == "widget_id" {
			t.Fatal("wire metadata must not leak into request fields")
		}
	}
}
