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

import "context"

// Field is one request-scoped key/value pair for the sinks, such as the
// request method or route. Attach fields to a context with ContextWithFields at
// the edge of a request; a Recorder or Logger reads them back with
// FieldsFromContext from the ctx it already receives, so the sink signatures do
// not change.
type Field struct {
	Key   string
	Value any
}

// fieldsKey is the unexported context key for request fields, so no other
// package can read or overwrite them by accident.
type fieldsKey struct{}

// ContextWithFields returns a child of ctx carrying fields after any fields ctx
// already has. Order is kept and duplicate keys are not merged: a sink sees
// every field in the order it was added and decides how to render repeats.
// The parent context is never modified, so sibling requests do not share
// fields. With no fields it returns ctx unchanged.
func ContextWithFields(ctx context.Context, fields ...Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	prev, _ := ctx.Value(fieldsKey{}).([]Field)
	// Build a fresh slice so appending on one child can never write into the
	// backing array another child shares with the parent.
	all := make([]Field, 0, len(prev)+len(fields))
	all = append(all, prev...)
	all = append(all, fields...)
	return context.WithValue(ctx, fieldsKey{}, all)
}

// FieldsFromContext returns the request fields attached to ctx, in the order
// they were added. It returns nil when ctx carries none (or is nil). The result
// is a copy, so a sink may modify it freely.
func FieldsFromContext(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}
	fields, _ := ctx.Value(fieldsKey{}).([]Field)
	if len(fields) == 0 {
		return nil
	}
	out := make([]Field, len(fields))
	copy(out, fields)
	return out
}
