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

import (
	"context"
	"errors"
	"maps"
	"strconv"
	"strings"

	apperr "github.com/Bugs5382/go-apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetaCodeNum is the ErrorInfo metadata key that always carries the numeric
// code. It overrides any wire metadata with the same key.
const MetaCodeNum = "codeNum"

// Code maps a category to its gRPC code. An unknown category maps to
// codes.Internal.
func Code(c apperr.Category) codes.Code {
	switch c {
	case apperr.CategoryNotFound:
		return codes.NotFound
	case apperr.CategoryInvalid:
		return codes.InvalidArgument
	case apperr.CategoryUnavailable:
		return codes.Unavailable
	case apperr.CategoryPermissionDenied:
		return codes.PermissionDenied
	case apperr.CategoryFailedPrecondition:
		return codes.FailedPrecondition
	case apperr.CategoryDeadlineExceeded:
		return codes.DeadlineExceeded
	case apperr.CategoryUnauthenticated:
		return codes.Unauthenticated
	case apperr.CategoryAlreadyExists:
		return codes.AlreadyExists
	default:
		return codes.Internal
	}
}

// Status presents err through reg and returns the gRPC status to send. It
// resolves the code exactly as Registry.PresentContext does (falling back to
// defaultCode for an uncoded error) and drives the registry's Recorder and
// Logger with ctx. The status code comes from the entry's Category, the
// message from Present, and an ErrorInfo detail carries the symbol (or code),
// domain and metadata. It returns nil for a nil err.
func Status(ctx context.Context, reg *apperr.Registry, err error, defaultCode int, domain string) *status.Status {
	if err == nil {
		return nil
	}
	msg, code := reg.PresentContext(ctx, err, defaultCode)
	entry, _ := reg.Describe(code)

	wire := apperr.Metadata(err)
	meta := make(map[string]string, len(wire)+1)
	for k, v := range wire {
		meta[validUTF8(k)] = validUTF8(v)
	}
	meta[MetaCodeNum] = strconv.Itoa(code)

	reason := entry.Symbol
	if reason == "" {
		reason = strconv.Itoa(code)
	}

	st := status.New(Code(entry.Category), msg)
	withInfo, detailErr := st.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: validUTF8(domain), Metadata: meta})
	if detailErr != nil {
		// Unreachable: WithDetails fails only for codes.OK, which Code never
		// returns, or for strings that are not valid UTF-8, which validUTF8 rules
		// out.
		return st
	}
	return withInfo
}

// Error is Status(...).Err(): the error a gRPC handler returns. It returns nil
// for a nil err.
func Error(ctx context.Context, reg *apperr.Registry, err error, defaultCode int, domain string) error {
	return Status(ctx, reg, err, defaultCode, domain).Err()
}

// Info is what FromStatus reads back from a status's ErrorInfo.
type Info struct {
	// Code is the numeric code from MetaCodeNum, or from a numeric Reason when
	// MetaCodeNum is absent. It is 0 when neither holds a number.
	Code int
	// Symbol is the ErrorInfo reason, or "" when the reason is the numeric
	// code (an entry with no Symbol). A valid Symbol never starts with a
	// digit, so the two never collide.
	Symbol string
	// Domain is the ErrorInfo domain.
	Domain string
	// Metadata is the wire metadata without MetaCodeNum, or nil when there is
	// none, matching apperr.Metadata on the sending side.
	Metadata map[string]string
}

// FromStatus reads the first ErrorInfo in st's details. It reports false when
// st is nil or carries no ErrorInfo.
func FromStatus(st *status.Status) (Info, bool) {
	if st == nil {
		return Info{}, false
	}
	for _, d := range st.Details() {
		ei, ok := d.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		return infoOf(ei), true
	}
	return Info{}, false
}

// FromError reads the ErrorInfo from a gRPC status error anywhere in err's
// chain, as a client receives it. It reports false when err holds no status or
// the status carries no ErrorInfo.
func FromError(err error) (Info, bool) {
	var se interface{ GRPCStatus() *status.Status }
	if !errors.As(err, &se) {
		return Info{}, false
	}
	return FromStatus(se.GRPCStatus())
}

// validUTF8 replaces invalid UTF-8, which protobuf refuses to marshal in a
// string field, so arbitrary metadata can never cost the ErrorInfo.
func validUTF8(s string) string {
	return strings.ToValidUTF8(s, "\uFFFD")
}

func infoOf(ei *errdetails.ErrorInfo) Info {
	info := Info{Symbol: ei.GetReason(), Domain: ei.GetDomain()}
	if n, err := strconv.Atoi(info.Symbol); err == nil {
		info.Code, info.Symbol = n, ""
	}
	meta := maps.Clone(ei.GetMetadata())
	if raw, ok := meta[MetaCodeNum]; ok {
		delete(meta, MetaCodeNum)
		if n, err := strconv.Atoi(raw); err == nil {
			info.Code = n
		}
	}
	if len(meta) > 0 {
		info.Metadata = meta
	}
	return info
}
