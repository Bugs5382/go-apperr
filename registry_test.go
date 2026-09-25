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

// TestWithServiceSingleDigitUnchanged pins the v1 behavior of a one-digit
// prefix: any code whose leading digit matches passes, whatever its length.
func TestWithServiceSingleDigitUnchanged(t *testing.T) {
	t.Parallel()

	for _, c := range []int{1, 12, 100, 1001, 19999, 123456, -1001} {
		if _, err := apperr.NewRegistry([]apperr.Entry{{Code: c}}, apperr.WithService(1)); err != nil {
			t.Errorf("code %d with WithService(1): unexpected error %v", c, err)
		}
	}
	for _, c := range []int{0, 2001, 9} {
		if _, err := apperr.NewRegistry([]apperr.Entry{{Code: c}}, apperr.WithService(1)); err == nil {
			t.Errorf("code %d with WithService(1): expected an error", c)
		}
	}
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 0}}, apperr.WithService(0)); err != nil {
		t.Errorf("code 0 with WithService(0): unexpected error %v", err)
	}
}

func TestWithServiceMultiDigitPrefix(t *testing.T) {
	t.Parallel()

	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 12001}, {Code: 12999}, {Code: 1200}}, apperr.WithService(12)); err != nil {
		t.Fatalf("codes starting with 12 should pass: %v", err)
	}
	for _, c := range []int{1001, 13001, 21001, 1, 102001} {
		if _, err := apperr.NewRegistry([]apperr.Entry{{Code: c}}, apperr.WithService(12)); err == nil {
			t.Errorf("code %d with WithService(12): expected an error", c)
		}
	}
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 123001}}, apperr.WithService(123)); err != nil {
		t.Fatalf("three-digit prefix: %v", err)
	}
}

func TestWithCodeDigitsDerivesRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix int
		digits int
		code   int
		ok     bool
	}{
		{"single digit low edge", 1, 4, 1000, true},
		{"single digit high edge", 1, 4, 1999, true},
		{"single digit too short", 1, 4, 100, false},
		{"single digit too long", 1, 4, 10000, false},
		{"two digit low edge", 12, 5, 12000, true},
		{"two digit high edge", 12, 5, 12999, true},
		{"two digit below range", 12, 5, 11999, false},
		{"two digit above range", 12, 5, 13000, false},
		{"two digit too short", 12, 5, 1200, false},
		{"two digit too long", 12, 5, 120001, false},
		{"three digit", 123, 6, 123456, true},
		{"three digit wrong prefix", 123, 6, 124456, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := apperr.NewRegistry([]apperr.Entry{{Code: tc.code}},
				apperr.WithService(tc.prefix), apperr.WithCodeDigits(tc.digits))
			if tc.ok && err != nil {
				t.Fatalf("code %d, prefix %d, %d digits: unexpected error %v", tc.code, tc.prefix, tc.digits, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("code %d, prefix %d, %d digits: expected an error", tc.code, tc.prefix, tc.digits)
			}
		})
	}
}

func TestWithCodeDigitsRejectsBadConfig(t *testing.T) {
	t.Parallel()

	// The width must leave room for at least one digit after the prefix.
	if _, err := apperr.NewRegistry(nil, apperr.WithService(12), apperr.WithCodeDigits(2)); err == nil {
		t.Fatal("a width equal to the prefix length should fail")
	}
	if _, err := apperr.NewRegistry(nil, apperr.WithCodeDigits(0)); err == nil {
		t.Fatal("a non-positive width should fail")
	}
}

func TestWithCodeDigitsWithoutService(t *testing.T) {
	t.Parallel()

	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001}, {Code: 9999}}, apperr.WithCodeDigits(4)); err != nil {
		t.Fatalf("four-digit codes should pass: %v", err)
	}
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 100}}, apperr.WithCodeDigits(4)); err == nil {
		t.Fatal("a three-digit code should fail a four-digit width")
	}
}

func TestMultiDigitPrefixStillRejectsDuplicate(t *testing.T) {
	t.Parallel()
	_, err := apperr.NewRegistry([]apperr.Entry{{Code: 12001}, {Code: 12001}},
		apperr.WithService(12), apperr.WithCodeDigits(5))
	if err == nil {
		t.Fatal("expected an error for a duplicate code")
	}
}
