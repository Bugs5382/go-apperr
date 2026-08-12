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
	"strings"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

func newTestRegistry(t *testing.T, opts ...apperr.Option) *apperr.Registry {
	t.Helper()
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "database", Cause: "database unavailable"},
		{Code: 1002, Title: "upstream", Cause: "upstream timeout"},
	}, opts...)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func TestNewRegistryRejectsDuplicate(t *testing.T) {
	t.Parallel()
	_, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "a"},
		{Code: 1001, Title: "b"},
	})
	if err == nil {
		t.Fatal("expected an error for a duplicate code")
	}
}

func TestWithServiceValidatesPrefix(t *testing.T) {
	t.Parallel()

	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001}, {Code: 1002}}, apperr.WithService(1)); err != nil {
		t.Fatalf("codes matching the service digit should pass: %v", err)
	}
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001}, {Code: 2001}}, apperr.WithService(1)); err == nil {
		t.Fatal("a code outside the service digit should fail")
	}
}

func TestPresentResolvesAndFallsBack(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)

	msg, code := reg.Present(apperr.Coded(1002, errors.New("i/o timeout")), 1001)
	if code != 1002 {
		t.Fatalf("code = %d; want 1002", code)
	}
	if !strings.Contains(msg, "1002") {
		t.Fatalf("message %q should mention the code", msg)
	}

	_, code = reg.Present(errors.New("uncoded"), 1001)
	if code != 1001 {
		t.Fatalf("fallback code = %d; want 1001", code)
	}
}

func TestMessageTemplate(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t, apperr.WithMessageTemplate("please quote ref %d"))
	if got := reg.Message(1001); got != "please quote ref 1001" {
		t.Fatalf("Message = %q", got)
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)

	e, ok := reg.Describe(1001)
	if !ok || e.Title != "database" {
		t.Fatalf("Describe(1001) = %+v,%t", e, ok)
	}
	if _, ok := reg.Describe(9999); ok {
		t.Fatal("Describe of an unknown code should report false")
	}
}

func TestMarkdownDeterministic(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)

	want := "| Code | Area | Cause |\n" +
		"| --- | --- | --- |\n" +
		"| 1001 | database | database unavailable |\n" +
		"| 1002 | upstream | upstream timeout |\n"
	if got := reg.Markdown(); got != want {
		t.Fatalf("Markdown() =\n%q\nwant\n%q", got, want)
	}
}

// recordingSink captures the last RecordCode/LogCoded call so the test can
// assert PresentContext drives the pluggable sinks.
type recordingSink struct {
	recordCalls int
	logCalls    int
	lastCode    int
}

func (s *recordingSink) RecordCode(_ context.Context, code int, _ error) {
	s.recordCalls++
	s.lastCode = code
}

func (s *recordingSink) LogCoded(_ context.Context, code int, _ error) {
	s.logCalls++
	s.lastCode = code
}

func TestPresentContextDrivesSinks(t *testing.T) {
	t.Parallel()
	sink := &recordingSink{}
	reg := newTestRegistry(t, apperr.WithRecorder(sink), apperr.WithLogger(sink))

	msg, code := reg.PresentContext(context.Background(), apperr.Coded(1002, errors.New("boom")), 1001)
	if code != 1002 || !strings.Contains(msg, "1002") {
		t.Fatalf("PresentContext = %q,%d", msg, code)
	}
	if sink.recordCalls != 1 || sink.logCalls != 1 || sink.lastCode != 1002 {
		t.Fatalf("sink = %+v; want one record + one log at code 1002", sink)
	}
}

func TestNilSinkOptionsAreIgnored(t *testing.T) {
	t.Parallel()
	// A nil sink must not overwrite the no-op default (no panic on use).
	reg := newTestRegistry(t, apperr.WithRecorder(nil), apperr.WithLogger(nil))
	if _, code := reg.PresentContext(context.Background(), errors.New("x"), 1001); code != 1001 {
		t.Fatalf("code = %d; want 1001", code)
	}
}
