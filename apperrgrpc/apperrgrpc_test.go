package apperrgrpc_test

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
	"fmt"
	"maps"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
	"github.com/Bugs5382/go-apperr/apperrgrpc"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const domain = "widgets.example.org"

func newRegistry(t *testing.T, opts ...apperr.Option) *apperr.Registry {
	t.Helper()
	reg, err := apperr.NewRegistry([]apperr.Entry{
		{Code: 6000, Title: "default", Cause: "unclassified"},
		{Code: 6001, Title: "resolver", Cause: "resolver down", Category: apperr.CategoryUnavailable},
		{Code: 6010, Title: "swap", Cause: "candidate not eligible", Category: apperr.CategoryFailedPrecondition,
			Symbol: "SWAP_NOT_ELIGIBLE", UserSafe: true, Message: "{new_user_id} is not eligible for stage {stage}."},
	}, opts...)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func errorInfo(t *testing.T, st *status.Status) *errdetails.ErrorInfo {
	t.Helper()
	for _, d := range st.Details() {
		if ei, ok := d.(*errdetails.ErrorInfo); ok {
			return ei
		}
	}
	t.Fatalf("status %v has no ErrorInfo", st)
	return nil
}

func TestCodeMapsEveryCategory(t *testing.T) {
	t.Parallel()
	tests := map[apperr.Category]codes.Code{
		apperr.CategoryInternal:           codes.Internal,
		apperr.CategoryNotFound:           codes.NotFound,
		apperr.CategoryInvalid:            codes.InvalidArgument,
		apperr.CategoryUnavailable:        codes.Unavailable,
		apperr.CategoryPermissionDenied:   codes.PermissionDenied,
		apperr.CategoryFailedPrecondition: codes.FailedPrecondition,
		apperr.CategoryDeadlineExceeded:   codes.DeadlineExceeded,
		apperr.CategoryUnauthenticated:    codes.Unauthenticated,
		apperr.CategoryAlreadyExists:      codes.AlreadyExists,
		apperr.Category(99):               codes.Internal,
	}
	for c, want := range tests {
		if got := apperrgrpc.Code(c); got != want {
			t.Errorf("Code(%v) = %v; want %v", c, got, want)
		}
	}
}

func TestStatusUserSafeWithSymbolAndMetadata(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	err := apperr.WithMeta(apperr.Coded(6010, errors.New("u-2 not in pool")),
		apperr.Meta("new_user_id", "u-2"), apperr.Meta("stage", "1"))

	st := apperrgrpc.Status(t.Context(), reg, err, 6000, domain)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %v; want FailedPrecondition", st.Code())
	}
	if want := "u-2 is not eligible for stage 1."; st.Message() != want {
		t.Fatalf("message = %q; want %q", st.Message(), want)
	}
	ei := errorInfo(t, st)
	if ei.GetReason() != "SWAP_NOT_ELIGIBLE" || ei.GetDomain() != domain {
		t.Fatalf("ErrorInfo reason/domain = %q/%q", ei.GetReason(), ei.GetDomain())
	}
	want := map[string]string{"new_user_id": "u-2", "stage": "1", apperrgrpc.MetaCodeNum: "6010"}
	if !maps.Equal(ei.GetMetadata(), want) {
		t.Fatalf("metadata = %v; want %v", ei.GetMetadata(), want)
	}
}

func TestStatusWithoutSymbolUsesCodeAsReason(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	st := apperrgrpc.Status(t.Context(), reg, apperr.Coded(6001, errors.New("dial tcp: refused")), 6000, domain)
	if st.Code() != codes.Unavailable || st.Message() != "Code 6001: Internal Error" {
		t.Fatalf("status = %v %q", st.Code(), st.Message())
	}
	ei := errorInfo(t, st)
	if ei.GetReason() != "6001" {
		t.Fatalf("reason = %q; want 6001", ei.GetReason())
	}
	if want := map[string]string{apperrgrpc.MetaCodeNum: "6001"}; !maps.Equal(ei.GetMetadata(), want) {
		t.Fatalf("metadata = %v; want %v", ei.GetMetadata(), want)
	}
}

func TestStatusUncodedFallsBackToDefault(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	st := apperrgrpc.Status(t.Context(), reg, errors.New("boom"), 6000, domain)
	if st.Code() != codes.Internal || st.Message() != "Code 6000: Internal Error" {
		t.Fatalf("status = %v %q", st.Code(), st.Message())
	}
	if got := errorInfo(t, st).GetMetadata()[apperrgrpc.MetaCodeNum]; got != "6000" {
		t.Fatalf("codeNum = %q; want 6000", got)
	}
}

func TestStatusUnregisteredCodeIsInternal(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	st := apperrgrpc.Status(t.Context(), reg, apperr.Coded(6999, nil), 6000, domain)
	if st.Code() != codes.Internal || errorInfo(t, st).GetReason() != "6999" {
		t.Fatalf("status = %v reason %q", st.Code(), errorInfo(t, st).GetReason())
	}
}

func TestStatusCodeNumWinsOverMetadata(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	err := apperr.WithMeta(apperr.Coded(6001, nil), apperr.Meta(apperrgrpc.MetaCodeNum, "1"))
	if got := errorInfo(t, apperrgrpc.Status(t.Context(), reg, err, 6000, domain)).GetMetadata()[apperrgrpc.MetaCodeNum]; got != "6001" {
		t.Fatalf("codeNum = %q; want 6001", got)
	}
}

func TestStatusAndErrorNilForNilError(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	if st := apperrgrpc.Status(t.Context(), reg, nil, 6000, domain); st != nil {
		t.Fatalf("Status(nil) = %v; want nil", st)
	}
	if err := apperrgrpc.Error(t.Context(), reg, nil, 6000, domain); err != nil {
		t.Fatalf("Error(nil) = %v; want nil", err)
	}
}

type countingLogger struct{ calls int }

func (l *countingLogger) LogCoded(context.Context, int, error) { l.calls++ }

func TestStatusDrivesSinks(t *testing.T) {
	t.Parallel()
	log := &countingLogger{}
	reg := newRegistry(t, apperr.WithLogger(log))
	_ = apperrgrpc.Error(t.Context(), reg, apperr.Coded(6001, nil), 6000, domain)
	if log.calls != 1 {
		t.Fatalf("logger calls = %d; want 1", log.calls)
	}
}

func TestRoundTripOverTheWire(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	err := apperr.WithMeta(apperr.Coded(6010, nil), apperr.Meta("new_user_id", "u-2"), apperr.Meta("stage", "1"))
	sent := apperrgrpc.Error(t.Context(), reg, err, 6000, domain)

	raw, mErr := proto.Marshal(status.Convert(sent).Proto())
	if mErr != nil {
		t.Fatal(mErr)
	}
	var pb spb.Status
	if uErr := proto.Unmarshal(raw, &pb); uErr != nil {
		t.Fatal(uErr)
	}
	received := fmt.Errorf("call widgets: %w", status.FromProto(&pb).Err())

	info, ok := apperrgrpc.FromError(received)
	if !ok {
		t.Fatal("FromError found no ErrorInfo")
	}
	want := apperrgrpc.Info{
		Code:     6010,
		Symbol:   "SWAP_NOT_ELIGIBLE",
		Domain:   domain,
		Metadata: map[string]string{"new_user_id": "u-2", "stage": "1"},
	}
	if info.Code != want.Code || info.Symbol != want.Symbol || info.Domain != want.Domain || !maps.Equal(info.Metadata, want.Metadata) {
		t.Fatalf("Info = %+v; want %+v", info, want)
	}
}

func TestFromStatusWithoutSymbol(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	info, ok := apperrgrpc.FromStatus(apperrgrpc.Status(t.Context(), reg, apperr.Coded(6001, nil), 6000, domain))
	if !ok || info.Code != 6001 || info.Symbol != "" || info.Metadata != nil {
		t.Fatalf("Info = %+v,%t; want code 6001, no symbol, nil metadata", info, ok)
	}
}

func withInfo(t *testing.T, ei *errdetails.ErrorInfo) *status.Status {
	t.Helper()
	st, err := status.New(codes.Internal, "x").WithDetails(ei)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestFromStatusForeignErrorInfo(t *testing.T) {
	t.Parallel()
	info, ok := apperrgrpc.FromStatus(withInfo(t, &errdetails.ErrorInfo{
		Reason: "QUOTA_EXCEEDED", Domain: "other.example.org", Metadata: map[string]string{"limit": "5"},
	}))
	if !ok || info.Code != 0 || info.Symbol != "QUOTA_EXCEEDED" || info.Domain != "other.example.org" || info.Metadata["limit"] != "5" {
		t.Fatalf("Info = %+v,%t", info, ok)
	}
}

func TestFromStatusNumericReasonWithoutCodeNum(t *testing.T) {
	t.Parallel()
	info, ok := apperrgrpc.FromStatus(withInfo(t, &errdetails.ErrorInfo{Reason: "6001", Domain: domain}))
	if !ok || info.Code != 6001 || info.Symbol != "" {
		t.Fatalf("Info = %+v,%t; want code 6001 and no symbol", info, ok)
	}
}

func TestFromStatusBadCodeNum(t *testing.T) {
	t.Parallel()
	info, ok := apperrgrpc.FromStatus(withInfo(t, &errdetails.ErrorInfo{
		Reason: "WIDGET_GONE", Metadata: map[string]string{apperrgrpc.MetaCodeNum: "abc"},
	}))
	if !ok || info.Code != 0 || info.Symbol != "WIDGET_GONE" || info.Metadata != nil {
		t.Fatalf("Info = %+v,%t", info, ok)
	}
}

func TestFromStatusSkipsOtherDetails(t *testing.T) {
	t.Parallel()
	st, err := status.New(codes.InvalidArgument, "bad").WithDetails(
		&errdetails.BadRequest{},
		&errdetails.ErrorInfo{Reason: "FIRST", Metadata: map[string]string{apperrgrpc.MetaCodeNum: "1"}},
		&errdetails.ErrorInfo{Reason: "SECOND", Metadata: map[string]string{apperrgrpc.MetaCodeNum: "2"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	info, ok := apperrgrpc.FromStatus(st)
	if !ok || info.Symbol != "FIRST" || info.Code != 1 {
		t.Fatalf("Info = %+v,%t; want the first ErrorInfo", info, ok)
	}
}

func TestFromStatusNoInfo(t *testing.T) {
	t.Parallel()
	if _, ok := apperrgrpc.FromStatus(nil); ok {
		t.Fatal("FromStatus(nil) should report false")
	}
	if _, ok := apperrgrpc.FromStatus(status.New(codes.NotFound, "missing")); ok {
		t.Fatal("a status without ErrorInfo should report false")
	}
	undecodable := status.FromProto(&spb.Status{Code: int32(codes.Internal), Details: []*anypb.Any{{TypeUrl: "type.googleapis.com/unknown.Type"}}})
	if _, ok := apperrgrpc.FromStatus(undecodable); ok {
		t.Fatal("an undecodable detail should be skipped")
	}
}

func TestFromErrorNonStatus(t *testing.T) {
	t.Parallel()
	if _, ok := apperrgrpc.FromError(errors.New("plain")); ok {
		t.Fatal("a plain error should report false")
	}
	if _, ok := apperrgrpc.FromError(nil); ok {
		t.Fatal("nil should report false")
	}
}

func TestStatusKeepsErrorInfoForInvalidUTF8(t *testing.T) {
	t.Parallel()
	reg := newRegistry(t)
	err := apperr.WithMeta(apperr.Coded(6001, nil), apperr.Meta("note\xff", "bad\xffvalue"))
	ei := errorInfo(t, apperrgrpc.Status(t.Context(), reg, err, 6000, "bad\xffdomain"))
	if ei.GetMetadata()["note\uFFFD"] != "bad\uFFFDvalue" || ei.GetDomain() != "bad\uFFFDdomain" {
		t.Fatalf("ErrorInfo = %v; want invalid bytes replaced", ei)
	}
	if ei.GetMetadata()[apperrgrpc.MetaCodeNum] != "6001" {
		t.Fatalf("codeNum = %q; want 6001", ei.GetMetadata()[apperrgrpc.MetaCodeNum])
	}
}
