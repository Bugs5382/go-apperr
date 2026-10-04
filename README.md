# go-apperr 🔢

> 🏷️ Turn internal errors into stable, reportable numeric **codes** — with logging and tracing you plug in, not baked in.

An internal error should not leak its guts to a client, and "internal error" with no handle is
useless in a bug report. `go-apperr` wraps a cause with a stable code that travels the error chain,
and at the edge renders a sanitized, quotable message. 🔌 **Bring your own logging and tracing:**
the package depends on **nothing** beyond the standard library.

## ✨ Highlights

- 🧬 **Codes ride the error chain** — `errors.Is`/`errors.As` keep working; recover the code anywhere.
- 🧼 **Sanitized at the edge** — clients get a code they can quote, never the raw internal detail.
- 🪶 **Zero dependencies** — the `go.mod` requires only the standard library.
- 🔧 **Pluggable observability** — tiny `Recorder`/`Logger` interfaces, no-op by default.
- 📝 **Docs-ready** — render your whole code table as Markdown.
- 🔤 **Symbols and user-safe messages** — a stable `SCREAMING_SNAKE` name per code, and an opt-in message end users may see.
- 📨 **Wire metadata** — key/value pairs on one error that travel to the client.
- 🛰️ **gRPC adapter** — `apperrgrpc`, its own module, builds a status with `ErrorInfo` and reads it back.

## 📦 Install

```bash
go get github.com/Bugs5382/go-apperr
```

No third-party dependencies come with it — you add a logger or tracer only if you wire one.

The gRPC adapter is a separate module, so only a service that imports it downloads gRPC:

```bash
go get github.com/Bugs5382/go-apperr/apperrgrpc
```

## 🚀 Core usage

Wrap a cause with a code; recover it anywhere with `Code`. `errors.Is`/`errors.As` still traverse
to the original cause.

```go
err := apperr.Coded(1001, fmt.Errorf("load widget: %w", errDatabaseDown))

code, ok := apperr.Code(err)      // 1001, true
errors.Is(err, errDatabaseDown)   // true
```

Register your codes once at startup, then present internal errors as client-safe messages:

```go
reg, err := apperr.NewRegistry([]apperr.Entry{
    {Code: 1001, Title: "database", Cause: "database unavailable"},
    {Code: 1002, Title: "upstream", Cause: "upstream timeout"},
}, apperr.WithService(1)) // this service owns the "1" prefix

msg, code := reg.Present(apperr.Coded(1002, dbErr), 1001)
// msg = "Code 1002: Internal Error", code = 1002
// an uncoded error falls back to the default code you pass.
```

A service prefix can be longer than one digit. `WithService(12)` accepts codes that start with
`12`, and `WithCodeDigits` fixes the code width so each prefix owns one range:

```go
reg, err := apperr.NewRegistry(entries,
    apperr.WithService(12),     // this service owns the "12" prefix
    apperr.WithCodeDigits(5),   // five-digit codes: 12000 through 12999
)
// 13001, 1201 and 120001 fail registration; so does a duplicate code.
```

A one-digit prefix keeps its original rule (any code whose first digit matches), so existing
registries keep their codes.

`WithMessageTemplate` overrides the client message; `Describe` looks a code back up for operators;
`Markdown` renders the whole registry as a `Code | Area | Cause` table (sorted by code) for your
error-codes doc. Once any entry sets a `Symbol` or `UserSafe`, the table becomes
`Code | Symbol | Area | Cause | User-safe`; a registry that uses neither renders exactly as before.

## 🔤 Symbols and user-safe messages

An entry can name its code with an optional `Symbol`, so clients and translation catalogs key on a
stable name instead of a number. A symbol must match `^[A-Z][A-Z0-9_]*$` and be unique in the
registry; `NewRegistry` rejects anything else.

`UserSafe` says the entry's `Message` may be shown to end users. `Present` then returns that
message instead of the generic template, filling each `{key}` placeholder from the error's wire
metadata (see below):

```go
reg, _ := apperr.NewRegistry([]apperr.Entry{
    {Code: 6001, Title: "resolver", Cause: "approver lookup failed"},
    {
        Code: 6010, Title: "swap", Cause: "candidate not eligible",
        Symbol: "SWAP_NOT_ELIGIBLE", Category: apperr.CategoryFailedPrecondition,
        UserSafe: true, Message: "That person isn't an eligible approver for stage {stage}.",
    },
})

msg, _ := reg.Present(apperr.WithMeta(apperr.Coded(6010, nil), apperr.Meta("stage", "2")), 6000)
// msg = "That person isn't an eligible approver for stage 2."

msg, _ = reg.Present(apperr.Coded(6001, dbErr), 6000)
// msg = "Code 6001: Internal Error"
```

🛡️ **Safe by default:** `UserSafe` is `false` unless an entry sets it, so every existing entry (and
any unregistered code) still gets only the template and its code. A user-safe entry must have a
non-blank `Message`. A `Message` on an entry that is not user-safe is never presented, but stays
available through `Describe` for your docs or UI catalog. A placeholder with no matching key is
left as written, and a substituted value is never scanned again. `Message(code)` always renders
the template; `Present` and `PresentContext` are the paths that honour `UserSafe`.

## 📨 Wire metadata

`WithMeta` attaches key/value pairs to one error, and `Metadata` reads them back from anywhere in
the chain (`errors.Join` trees included; an outer layer wins on a repeated key):

```go
err := apperr.WithMeta(
    apperr.Coded(6010, fmt.Errorf("swap: %w", errNotEligible)),
    apperr.Meta("new_user_id", candidateID),
)

apperr.Metadata(err) // map[new_user_id:...]
```

The wrapped error keeps its `Error()` text, its code and its `errors.Is`/`errors.As` behaviour.

📌 **Metadata is not Fields.** They answer different questions:

| | Wire metadata (`WithMeta`) | Request fields (`ContextWithFields`) |
| --- | --- | --- |
| Describes | this one failure | the request |
| Lives on | the error | the context |
| Reaches | the client (status details, message placeholders) | the `Recorder` and `Logger` only |

Never put anything in metadata the client may not see; log-only detail belongs in Fields or the
wrapped cause.

## 🧭 Map to a transport

An entry can carry an optional, transport-neutral `Category`: `CategoryInternal` (the default when
unset), `CategoryNotFound`, `CategoryInvalid`, `CategoryUnavailable`, `CategoryPermissionDenied`,
`CategoryFailedPrecondition`, `CategoryDeadlineExceeded`, `CategoryUnauthenticated` or
`CategoryAlreadyExists`. New categories are only ever appended, so no constant changes value.
`reg.Category(err)` looks it up from a coded error, and falls back to internal for an uncoded or
unregistered one. One small mapper per transport then covers every code:

```go
reg, _ := apperr.NewRegistry([]apperr.Entry{
    {Code: 1002, Title: "widget", Cause: "widget not found", Category: apperr.CategoryNotFound},
})

// gRPC, in your server package
func grpcCode(c apperr.Category) codes.Code {
    switch c {
    case apperr.CategoryNotFound:
        return codes.NotFound
    case apperr.CategoryInvalid:
        return codes.InvalidArgument
    case apperr.CategoryUnavailable:
        return codes.Unavailable
    case apperr.CategoryPermissionDenied:
        return codes.PermissionDenied
    default:
        return codes.Internal
    }
}

msg, _ := reg.Present(err, 1000)
return status.Error(grpcCode(reg.Category(err)), msg)

// HTTP: the same switch returning http.StatusNotFound, http.StatusBadRequest,
// http.StatusServiceUnavailable, http.StatusForbidden or http.StatusInternalServerError
http.Error(w, msg, httpStatus(reg.Category(err)))
```

`go-apperr` itself imports neither transport; the mappers live in your code. The `Markdown` table
is unchanged by categories, so existing error-codes docs stay in sync.

### gRPC adapter

For gRPC you can skip the hand-written switch: `apperrgrpc` maps every category (`Code`) and builds
the whole status. The status carries one `errdetails.ErrorInfo` whose `Reason` is the entry's
`Symbol` (or the code when it has none), whose `Domain` is the one you pass, and whose `Metadata`
is the error's wire metadata plus the numeric code under `codeNum`:

```go
import "github.com/Bugs5382/go-apperr/apperrgrpc"

// server: present through the registry (sinks included) and return the status error
return nil, apperrgrpc.Error(ctx, reg, err, 6000, "workflow.example.org")

// client: read it back
if info, ok := apperrgrpc.FromError(err); ok {
    // info.Code == 6010, info.Symbol == "SWAP_NOT_ELIGIBLE",
    // info.Domain == "workflow.example.org", info.Metadata["new_user_id"] == candidateID
}
```

`codeNum` always rides along, so a relaying service recovers the original code without a copy of
the remote registry. `FromStatus` does the same for a `*status.Status`. The adapter is its own Go
module (`github.com/Bugs5382/go-apperr/apperrgrpc`), so the root module keeps zero dependencies.

## 🔌 Bring your own logging and tracing

The package calls two tiny interfaces, defaulting to no-ops:

```go
type Recorder interface { RecordCode(ctx context.Context, code int, err error) }
type Logger   interface { LogCoded(ctx context.Context, code int, err error) }
```

Wire them with `WithRecorder`/`WithLogger`; `PresentContext` drives them while returning the same
sanitized message and code. On any stack, an adapter is a few lines — here a `log/slog` logger:

```go
type slogLogger struct{ log *slog.Logger }

func (s slogLogger) LogCoded(ctx context.Context, code int, err error) {
    s.log.ErrorContext(ctx, "coded error", "code", code, "error", err)
}

reg, _ := apperr.NewRegistry(entries, apperr.WithLogger(slogLogger{log: mySlog}))
```

To get request metadata (method, route, and so on) into the sinks, attach it to the context once
at the edge. A sink reads it back from the `ctx` it already receives, so the interfaces stay the
same:

```go
ctx = apperr.ContextWithFields(ctx,
    apperr.Field{Key: "method", Value: r.Method},
    apperr.Field{Key: "route", Value: route},
)
msg, code := reg.PresentContext(ctx, err, 1001)

// inside a sink
for _, f := range apperr.FieldsFromContext(ctx) {
    args = append(args, f.Key, f.Value)
}
```

Fields keep the order they were added, and `FieldsFromContext` returns nil when there are none.

A `Recorder` plugs in the same way — implement `RecordCode` over your tracer (OpenTelemetry, etc.)
to set the code on the active span. Nothing is bundled, so **your `go.mod` stays free of any
dependency you did not choose.** 🎯

## 📚 Examples

Runnable programs under [`examples/`](examples): `coded`, `registry`, `markdown`, and `custom`
(bring-your-own sinks over `log/slog`).

## 📐 Where this belongs

Use `go-apperr` in **application code** — the service that owns its code namespace and presents
errors at the edge. Don't depend on it from another reusable library: libraries should return
plain wrapped errors and let the app assign codes. Each app owns its own prefix (`WithService`) so
a code is attributable at a glance.

## 🛠 Develop

```bash
task build    # go build ./... (root module and apperrgrpc)
task test     # go test ./... (root module and apperrgrpc)
task lint     # gofmt + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
