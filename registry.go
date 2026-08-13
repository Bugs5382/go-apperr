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

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// defaultMessageTemplate is the sanitized, client-facing message rendered for a
// code when the consumer sets no WithMessageTemplate. It reveals only the code,
// never the internal cause.
const defaultMessageTemplate = "Code %d: Internal Error"

// Entry describes one code the consumer registers. Title is a short area/label
// for the code; Cause is a one-line internal description. Both are for the
// consumer's own error-codes documentation (see Markdown) and are never shown
// to a client -- only the code and the rendered message template are.
type Entry struct {
	Code  int
	Title string
	Cause string
}

// Registry is the consumer's own set of codes plus the presentation and
// observability policy around them. Build one at startup with NewRegistry and
// share it; it is read-only afterward and safe for concurrent use.
type Registry struct {
	entries  map[int]Entry
	order    []int
	service  int
	hasSvc   bool
	tmpl     string
	recorder Recorder
	logger   Logger
}

// Option configures a Registry at construction. All options are optional; the
// zero configuration is a plain code table with the default message template
// and no-op observability.
type Option func(*Registry)

// WithService validates that every registered code's first digit equals digit.
// It expresses the "each service owns a code prefix" convention: an app claims
// a leading digit so any code it emits is attributable to it at a glance.
func WithService(digit int) Option {
	return func(r *Registry) {
		r.service = digit
		r.hasSvc = true
	}
}

// WithMessageTemplate overrides the sanitized client message. tmpl is a
// fmt-style template that receives the code (include a %d), e.g.
// "Something went wrong (ref %d)". The default is "Code %d: Internal Error".
func WithMessageTemplate(tmpl string) Option {
	return func(r *Registry) { r.tmpl = tmpl }
}

// WithRecorder installs a tracing sink invoked by PresentContext. A nil
// Recorder is ignored, leaving the no-op default in place.
func WithRecorder(rec Recorder) Option {
	return func(r *Registry) {
		if rec != nil {
			r.recorder = rec
		}
	}
}

// WithLogger installs a logging sink invoked by PresentContext. A nil Logger is
// ignored, leaving the no-op default in place.
func WithLogger(l Logger) Option {
	return func(r *Registry) {
		if l != nil {
			r.logger = l
		}
	}
}

// NewRegistry builds a Registry from the consumer's codes and options. It fails
// on a duplicate code, and -- when WithService is set -- on any code whose
// first digit does not match the claimed service digit.
func NewRegistry(entries []Entry, opts ...Option) (*Registry, error) {
	r := &Registry{
		entries:  make(map[int]Entry, len(entries)),
		tmpl:     defaultMessageTemplate,
		recorder: nopRecorder{},
		logger:   nopLogger{},
	}
	for _, o := range opts {
		o(r)
	}
	for _, e := range entries {
		if _, dup := r.entries[e.Code]; dup {
			return nil, fmt.Errorf("apperr: duplicate code %d", e.Code)
		}
		if r.hasSvc {
			if d := firstDigit(e.Code); d != r.service {
				return nil, fmt.Errorf("apperr: code %d starts with %d, want service digit %d", e.Code, d, r.service)
			}
		}
		r.entries[e.Code] = e
	}
	r.order = make([]int, 0, len(r.entries))
	for c := range r.entries {
		r.order = append(r.order, c)
	}
	sort.Ints(r.order)
	return r, nil
}

// Describe returns the registered Entry for code and whether it was found.
func (r *Registry) Describe(code int) (Entry, bool) {
	e, ok := r.entries[code]
	return e, ok
}

// Message renders the sanitized client message for code via the template. It
// works for any code, registered or not, so an unexpected code still yields a
// safe, quotable message rather than leaking internals.
func (r *Registry) Message(code int) string {
	return fmt.Sprintf(r.tmpl, code)
}

// Present resolves the code from err (the nearest coded error) or falls back to
// defaultCode when err carries none, and returns the sanitized client message
// and the resolved code. It has no side effects; use PresentContext to also
// drive the Recorder and Logger.
func (r *Registry) Present(err error, defaultCode int) (clientMsg string, code int) {
	code = defaultCode
	if c, ok := Code(err); ok {
		code = c
	}
	return r.Message(code), code
}

// PresentContext is Present plus observability: after resolving the code it
// calls the configured Recorder and Logger with ctx, the code, and err. With
// the default no-op sinks it behaves exactly like Present.
func (r *Registry) PresentContext(ctx context.Context, err error, defaultCode int) (clientMsg string, code int) {
	clientMsg, code = r.Present(err, defaultCode)
	r.recorder.RecordCode(ctx, code, err)
	r.logger.LogCoded(ctx, code, err)
	return clientMsg, code
}

// Markdown renders the registry as a "Code | Area | Cause" table, sorted by
// code so the output is deterministic. It is meant to be written into a
// consumer's error-codes reference document.
func (r *Registry) Markdown() string {
	var b strings.Builder
	b.WriteString("| Code | Area | Cause |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, c := range r.order {
		e := r.entries[c]
		fmt.Fprintf(&b, "| %d | %s | %s |\n", e.Code, mdCell(e.Title), mdCell(e.Cause))
	}
	return b.String()
}

// firstDigit returns the leading decimal digit of code's absolute value.
func firstDigit(code int) int {
	n := code
	if n < 0 {
		n = -n
	}
	for n >= 10 {
		n /= 10
	}
	return n
}

// mdCell makes a string safe to drop into a Markdown table cell by escaping the
// pipe and flattening newlines, so a Title or Cause never breaks the table.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
