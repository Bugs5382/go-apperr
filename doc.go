// Package apperr turns internal errors into stable, reportable numeric codes.
//
// A service wraps an internal cause with Coded(code, cause). The code travels
// with the error through the normal error chain, so errors.Is and errors.As
// keep working, and Code(err) recovers the nearest code at any boundary. At the
// edge, a Registry the consumer builds at startup maps each code to a sanitized,
// client-safe message: the caller learns a code they can quote in a bug report,
// never the raw internal detail.
//
// The package is dependency-free: it imports no logging, tracing, or
// third-party package, only the standard library. Observability is pluggable
// through two small interfaces, Recorder and Logger, wired with WithRecorder
// and WithLogger and defaulting to no-ops. Bring any stack -- log/slog,
// OpenTelemetry, zap -- by writing a few-line adapter that implements them (see
// the examples). Nothing is bundled, so a consumer's go.mod stays free of any
// dependency they did not choose.
//
// Codes are the consumer's own namespace. The service prefix convention
// (WithService) is one supported way to let each service own a code prefix so a
// code is easy to attribute at a glance; it is optional. A prefix may be one
// digit or several, and WithCodeDigits fixes the code width so each prefix owns
// a single range (WithService(12) with WithCodeDigits(5) owns 12000-12999).
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
