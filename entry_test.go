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
	"strings"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
)

func TestSymbolAccepted(t *testing.T) {
	t.Parallel()
	for _, sym := range []string{"A", "WIDGET_NOT_FOUND", "V2_LIMIT", "TRAILING_"} {
		if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, Symbol: sym}}); err != nil {
			t.Errorf("symbol %q: %v", sym, err)
		}
	}
}

func TestSymbolRejectedWhenMalformed(t *testing.T) {
	t.Parallel()
	for _, sym := range []string{"widget", "Widget_Not_Found", "1WIDGET", "_WIDGET", "WIDGET-GONE", "WIDGET GONE"} {
		_, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, Symbol: sym}})
		if err == nil || !strings.Contains(err.Error(), sym) {
			t.Errorf("symbol %q: err = %v; want a rejection naming it", sym, err)
		}
	}
}

func TestSymbolRejectedWhenDuplicate(t *testing.T) {
	t.Parallel()
	_, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Symbol: "WIDGET_GONE"},
		{Code: 1002, Symbol: "WIDGET_GONE"},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate symbol") {
		t.Fatalf("err = %v; want a duplicate symbol error", err)
	}
}

func TestSymbolOptionalAndEmptyNotDuplicate(t *testing.T) {
	t.Parallel()
	if _, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001}, {Code: 1002}}); err != nil {
		t.Fatalf("entries without symbols: %v", err)
	}
}

func TestDescribeCarriesNewFields(t *testing.T) {
	t.Parallel()
	in := apperr.Entry{Code: 1001, Symbol: "WIDGET_GONE", UserSafe: true, Message: "That widget is gone."}
	reg, err := apperr.NewRegistry([]apperr.Entry{in})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := reg.Describe(1001); !ok || got != in {
		t.Fatalf("Describe = %+v,%t; want %+v", got, ok, in)
	}
}

func TestUserSafeRequiresMessage(t *testing.T) {
	t.Parallel()
	for _, msg := range []string{"", "   "} {
		_, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, UserSafe: true, Message: msg}})
		if err == nil || !strings.Contains(err.Error(), "1001") {
			t.Errorf("message %q: err = %v; want a rejection naming the code", msg, err)
		}
	}
}

func newSafeRegistry(t *testing.T, opts ...apperr.Option) *apperr.Registry {
	t.Helper()
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1001, Title: "database", Cause: "database unavailable", Message: "The database is resting."},
		{Code: 1002, Title: "widget", Cause: "widget not found", UserSafe: true, Message: "That widget no longer exists."},
		{Code: 1003, Title: "swap", Cause: "candidate not eligible", UserSafe: true,
			Message: "{new_user_id} can't approve stage {stage} ({missing})."},
	}, opts...)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func TestPresentUserSafeMessage(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t)
	msg, code := reg.Present(apperr.Coded(1002, errors.New("sql: no rows")), 1000)
	if msg != "That widget no longer exists." || code != 1002 {
		t.Fatalf("Present = %q,%d", msg, code)
	}
}

func TestPresentNotUserSafeKeepsTemplate(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t, apperr.WithMessageTemplate("ref %d"))
	if msg, _ := reg.Present(apperr.Coded(1001, nil), 1000); msg != "ref 1001" {
		t.Fatalf("not user-safe: msg = %q; want the template", msg)
	}
	if msg, _ := reg.Present(apperr.Coded(4242, nil), 1000); msg != "ref 4242" {
		t.Fatalf("unregistered: msg = %q; want the template", msg)
	}
}

func TestPresentUserSafeDefaultCode(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t)
	msg, code := reg.Present(errors.New("uncoded"), 1002)
	if msg != "That widget no longer exists." || code != 1002 {
		t.Fatalf("Present = %q,%d", msg, code)
	}
}

func TestPresentInterpolatesMetadata(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t)
	err := apperr.WithMeta(apperr.Coded(1003, nil),
		apperr.Meta("new_user_id", "{stage}"), apperr.Meta("stage", "2"))
	msg, _ := reg.Present(err, 1000)
	if want := "{stage} can't approve stage 2 ({missing})."; msg != want {
		t.Fatalf("msg = %q; want %q", msg, want)
	}
}

func TestPresentInterpolationEdgeCases(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"no placeholders":   "no placeholders",
		"open { only":       "open { only",
		"{} empty":          "{} empty",
		"{k}{k}":            "vv",
		"{{k}}":             "{v}",
		"tail {k":           "tail {k",
		"{unknown} and {k}": "{unknown} and v",
	}
	for tmpl, want := range tests {
		reg, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, UserSafe: true, Message: tmpl}})
		if err != nil {
			t.Fatal(err)
		}
		msg, _ := reg.Present(apperr.WithMeta(apperr.Coded(1001, nil), apperr.Meta("k", "v")), 1000)
		if msg != want {
			t.Errorf("template %q: msg = %q; want %q", tmpl, msg, want)
		}
	}
}

func TestPresentContextUsesUserSafeMessage(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t)
	msg, code := reg.PresentContext(t.Context(), apperr.Coded(1002, nil), 1000)
	if msg != "That widget no longer exists." || code != 1002 {
		t.Fatalf("PresentContext = %q,%d", msg, code)
	}
}

func TestMessageStaysTemplate(t *testing.T) {
	t.Parallel()
	reg := newSafeRegistry(t)
	if got := reg.Message(1002); got != "Code 1002: Internal Error" {
		t.Fatalf("Message(1002) = %q; want the template", got)
	}
}

func TestMarkdownAddsColumnsWhenUsed(t *testing.T) {
	t.Parallel()
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1002, Title: "widget", Cause: "widget not found", Symbol: "WIDGET_NOT_FOUND", UserSafe: true, Message: "Gone."},
		{Code: 1001, Title: "database", Cause: "database unavailable"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "| Code | Symbol | Area | Cause | User-safe |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| 1001 |  | database | database unavailable | no |\n" +
		"| 1002 | `WIDGET_NOT_FOUND` | widget | widget not found | yes |\n"
	if got := reg.Markdown(); got != want {
		t.Fatalf("Markdown() =\n%s\nwant\n%s", got, want)
	}
}

func TestMarkdownAddsColumnsForUserSafeAlone(t *testing.T) {
	t.Parallel()
	reg, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, Title: "a", Cause: "b", UserSafe: true, Message: "m"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := reg.Markdown(); !strings.HasPrefix(got, "| Code | Symbol | Area | Cause | User-safe |\n") {
		t.Fatalf("Markdown() = %q; want the extended header", got)
	}
}

func TestMarkdownUnchangedByMessageAlone(t *testing.T) {
	t.Parallel()
	reg, err := apperr.NewRegistry([]apperr.Entry{{Code: 1001, Title: "database", Cause: "database unavailable", Message: "Resting."}})
	if err != nil {
		t.Fatal(err)
	}
	want := "| Code | Area | Cause |\n| --- | --- | --- |\n| 1001 | database | database unavailable |\n"
	if got := reg.Markdown(); got != want {
		t.Fatalf("Markdown() = %q; want %q", got, want)
	}
}
