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

## 📦 Install

```bash
go get github.com/Bugs5382/go-apperr
```

No third-party dependencies come with it — you add a logger or tracer only if you wire one.

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

`WithMessageTemplate` overrides the client message; `Describe` looks a code back up for operators;
`Markdown` renders the whole registry as a `Code | Area | Cause` table (sorted by code) for your
error-codes doc.

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
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
