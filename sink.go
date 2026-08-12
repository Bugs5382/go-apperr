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

// Recorder is the neutral tracing sink the observability-aware present path
// calls. An implementation records the resolved code (and, if it wants, the
// error) on the active span or whatever backend it wraps. This package ships
// only a no-op; a real Recorder is a few lines of the consumer's own code over
// their tracer of choice (OpenTelemetry, etc.) -- see the examples.
type Recorder interface {
	RecordCode(ctx context.Context, code int, err error)
}

// Logger is the neutral logging sink the observability-aware present path
// calls. An implementation emits one structured line carrying the code and,
// when the backend correlates traces, the trace_id. As with Recorder, this
// package ships only a no-op; a small custom shim over the consumer's logger
// (log/slog, zap, etc.) provides the real thing.
type Logger interface {
	LogCoded(ctx context.Context, code int, err error)
}

// nopRecorder is the default Recorder: it records nothing, so a Registry with
// no WithRecorder option has zero observability overhead and no dependency.
type nopRecorder struct{}

func (nopRecorder) RecordCode(context.Context, int, error) {}

// nopLogger is the default Logger: it logs nothing, for the same reason.
type nopLogger struct{}

func (nopLogger) LogCoded(context.Context, int, error) {}
