# AGENTS.md - go-apperr

Guide for AI agents working in this repository. Pair with `CLAUDE.md` (the working agreement and
hook-enforced rules). Keep this file current when the build, layout, or public API changes.

## What this is

A generic coded-error library. A service wraps an internal cause with `apperr.Coded(code, cause)`;
the code rides along the normal error chain (`Unwrap`, so `errors.Is`/`errors.As` keep working) and
`apperr.Code(err)` recovers the nearest code at a boundary. At the edge, a `Registry` the consumer
builds at startup maps each code to a sanitized, client-safe message -- the caller gets a code they
can quote, never the raw internal detail.

Logging and tracing are **pluggable**. The package defines two tiny neutral interfaces, `Recorder`
(tracing) and `Logger` (logging), wired with `WithRecorder`/`WithLogger` and defaulting to no-ops.
`PresentContext` drives them. Bring any stack (log/slog, OpenTelemetry, zap, ...) by writing a
few-line adapter that implements the interfaces -- nothing is bundled, and the package imports no
logging, tracing, or third-party dependency.

## Where to use it (important)

`go-apperr` belongs in **application code** -- the service that owns the code namespace and the
edge where errors are presented. Do **not** take a dependency on it from another reusable library:
a library that forces a coded-error convention on its callers (and can end up in an import cycle
with the app that also uses `go-apperr`) is the wrong layer. Libraries should return plain wrapped
errors; the app assigns codes. Each app owns its own code prefix (see `WithService`) so a code is
attributable at a glance.

## Public API (`package apperr`, dependency-free)

- Errors: `Coded`, `Code`.
- Registry: `Entry`, `Registry`, `NewRegistry`, and methods `Describe`, `Message`, `Present`,
  `PresentContext`, `Markdown`.
- Options: `WithService`, `WithMessageTemplate`, `WithRecorder`, `WithLogger`.
- Sinks: the neutral `Recorder` and `Logger` interfaces (default no-op).

## Layout

Single module, `github.com/Bugs5382/go-apperr` (stdlib only -- `go list -m all` shows just this
module):

- `doc.go` - package overview.
- `apperr.go` - `Coded`, the internal `codedError`, `Code`.
- `sink.go` - the `Recorder`/`Logger` interfaces and their no-op defaults.
- `registry.go` - `Entry`, `Registry`, options, `NewRegistry`, and the present/describe/message/
  markdown methods.
- `*_test.go`, `example_test.go` - unit tests and verified `Example` functions.
- `examples/` - runnable `package main` programs: `coded`, `registry`, `markdown`, and `custom`
  (bring-your-own sinks over `log/slog`). They import only the core package and the standard
  library, so the module stays dependency-free.

## Build, test, lint

- Build: `task build` (`go build ./...`)
- Test: `task test` (`go test ./...`); no external service/fixture required.
- Lint: `task lint` (gofmt check + `golangci-lint run` + `yamllint .`)
- Full local gate: `task ci` (build + `go vet` + test + lint)
- License headers: `task license` (check) / `task license:fix` (inject)

## Conventions and gotchas

- See `CLAUDE.md` for the branch/commit/PR rules; they are enforced by the git hooks in
  `.claude/hooks` (run `bash .claude/hooks/install.sh` once per clone).
- Keep the package dependency-free: it must import no logging, tracing, or third-party package.
  Observability backends are the consumer's own adapter, never added here.
- Any change to `Recorder`, `Logger`, or an exported registry method is a public-API change: keep
  it additive unless a change is explicitly scoped as a major bump, and update this file and the
  README when the surface changes.
- `WithMessageTemplate` takes a fmt template that receives the code (include a `%d`); the default
  is `"Code %d: Internal Error"`.
