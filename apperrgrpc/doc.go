// Package apperrgrpc carries go-apperr coded errors across gRPC.
//
// It maps an apperr.Category to a gRPC code, and turns a coded error into a
// *status.Status whose details hold one errdetails.ErrorInfo:
//
//   - Reason is the entry's Symbol, or the numeric code when the entry has none;
//   - Domain is the domain the caller passes, usually the service's name;
//   - Metadata is the error's wire metadata (apperr.WithMeta) plus the numeric
//     code under MetaCodeNum, so a relaying process recovers the original code
//     without a copy of the remote registry.
//
// The status message is what Registry.Present returns: a user-safe entry's
// message, or the generic template. FromStatus and FromError read an ErrorInfo
// back into an Info.
//
// This package is its own Go module, so the gRPC dependency stays out of the
// root go-apperr module: a service that does not import apperrgrpc never
// downloads gRPC.
package apperrgrpc

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
