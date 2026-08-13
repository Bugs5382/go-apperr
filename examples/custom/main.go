// Command custom shows "bring your own" observability. The sinks here depend on
// nothing beyond the standard library: the logger is a few lines over log/slog,
// and the recorder is a trivial tally. Wire OpenTelemetry, zap, or any backend
// the same way -- each sink is a one-method interface.
package main

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
	"log/slog"
	"os"

	apperr "github.com/Bugs5382/go-apperr"
)

// slogLogger implements apperr.Logger over log/slog.
type slogLogger struct{ log *slog.Logger }

func (s slogLogger) LogCoded(ctx context.Context, code int, err error) {
	s.log.ErrorContext(ctx, "coded error", "code", code, "error", err)
}

// countingRecorder implements apperr.Recorder. A real one would set an
// attribute on the active tracing span; this one just tallies by code, to keep
// the example dependency-free while showing where a recorder plugs in.
type countingRecorder struct{ byCode map[int]int }

func (r *countingRecorder) RecordCode(_ context.Context, code int, _ error) {
	r.byCode[code]++
}

func main() {
	rec := &countingRecorder{byCode: map[int]int{}}
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 1002, Title: "upstream", Cause: "upstream timeout"},
	},
		apperr.WithRecorder(rec),
		apperr.WithLogger(slogLogger{log: slog.New(slog.NewJSONHandler(os.Stderr, nil))}),
	)
	if err != nil {
		panic(err)
	}

	internal := apperr.Coded(1002, errors.New("dial upstream: i/o timeout"))
	// PresentContext drives both sinks; the returned client message is what a
	// handler would send back, dropped here for brevity.
	_, _ = reg.PresentContext(context.Background(), internal, 1000)
}
