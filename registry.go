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
	"strconv"
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
	digits   int
	hasWidth bool
	tmpl     string
	recorder Recorder
	logger   Logger
}

// Option configures a Registry at construction. All options are optional; the
// zero configuration is a plain code table with the default message template
// and no-op observability.
type Option func(*Registry)

// WithService validates that every registered code starts with the decimal
// digits of prefix. It expresses the "each service owns a code prefix"
// convention: an app claims a prefix so any code it emits is attributable to it
// at a glance.
//
// A one-digit prefix behaves as it always has: WithService(1) accepts any code
// whose first digit is 1 (1, 12, 1001, ...). A longer prefix works the same way
// on more digits, so WithService(12) accepts 12001 and rejects 1001 and 13001.
// The sign of a code is ignored. Pair it with WithCodeDigits to pin every code
// to one fixed-width range derived from the prefix length.
func WithService(prefix int) Option {
	return func(r *Registry) {
		r.service = prefix
		r.hasSvc = true
	}
}

// WithCodeDigits requires every registered code to have exactly n decimal
// digits (sign ignored). With WithService it fixes each service's code range
// from the prefix length: a prefix of length L owns the n-L trailing digits, so
// WithService(12) with WithCodeDigits(5) accepts 12000 through 12999 and
// rejects anything outside it. NewRegistry fails when n is not positive or
// leaves no digit after the prefix.
func WithCodeDigits(n int) Option {
	return func(r *Registry) {
		r.digits = n
		r.hasWidth = true
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
// on a duplicate code, on any code outside the service prefix when WithService
// is set, and on any code of the wrong width when WithCodeDigits is set.
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
	if err := r.checkLayout(); err != nil {
		return nil, err
	}
	for _, e := range entries {
		if _, dup := r.entries[e.Code]; dup {
			return nil, fmt.Errorf("apperr: duplicate code %d", e.Code)
		}
		if err := r.checkCode(e.Code); err != nil {
			return nil, err
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

// checkLayout validates the prefix and width options against each other before
// any code is checked, so a misconfigured registry fails even with no entries.
func (r *Registry) checkLayout() error {
	if !r.hasWidth {
		return nil
	}
	if r.digits < 1 {
		return fmt.Errorf("apperr: code width %d must be positive", r.digits)
	}
	if r.hasSvc {
		if l := len(digitsOf(r.service)); l >= r.digits {
			return fmt.Errorf("apperr: code width %d leaves no digits after service prefix %d", r.digits, r.service)
		}
	}
	return nil
}

// checkCode applies the service prefix and code width rules to one code.
func (r *Registry) checkCode(code int) error {
	ds := digitsOf(code)
	if r.hasSvc {
		// Compare the prefix as a string of digits. For a one-digit prefix this is
		// exactly the original first-digit rule, and it extends to any length. A
		// negative prefix never matches, as before.
		want := strconv.Itoa(r.service)
		if !strings.HasPrefix(ds, want) {
			if len(want) == 1 {
				return fmt.Errorf("apperr: code %d starts with %c, want service digit %d", code, ds[0], r.service)
			}
			return fmt.Errorf("apperr: code %d does not start with service prefix %d", code, r.service)
		}
	}
	if r.hasWidth && len(ds) != r.digits {
		if r.hasSvc {
			lo, hi := r.serviceRange()
			return fmt.Errorf("apperr: code %d is outside service %d range %d-%d", code, r.service, lo, hi)
		}
		return fmt.Errorf("apperr: code %d has %d digits, want %d", code, len(ds), r.digits)
	}
	return nil
}

// serviceRange returns the inclusive code range a service prefix owns under the
// configured width. checkLayout guarantees the width exceeds the prefix length.
func (r *Registry) serviceRange() (lo, hi int) {
	span := 1
	for range r.digits - len(digitsOf(r.service)) {
		span *= 10
	}
	lo = r.service * span
	return lo, lo + span - 1
}

// digitsOf returns the decimal digits of code's absolute value.
func digitsOf(code int) string {
	return strings.TrimPrefix(strconv.Itoa(code), "-")
}

// mdCell makes a string safe to drop into a Markdown table cell by escaping the
// pipe and flattening newlines, so a Title or Cause never breaks the table.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
