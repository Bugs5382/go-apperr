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

## Public API (`package apperr`, standard library only)

- Errors: `Coded`, `Code`; wire metadata `WithMeta`, `Meta`, `MetaPair`, `Metadata`.
- Registry: `Entry` (with `Category`, `Symbol`, `UserSafe`, `Message`), `Registry`, `NewRegistry`,
  and methods `Describe`, `Message`, `Present`, `PresentContext`, `Category`, `Markdown`.
- Categories: `Category` and its constants. Only ever append new ones; never renumber.
- Options: `WithService`, `WithMessageTemplate`, `WithRecorder`, `WithLogger`.
- Sinks: the neutral `Recorder` and `Logger` interfaces (default no-op), and request `Field`s
  (`ContextWithFields`, `FieldsFromContext`).

## gRPC adapter (`apperrgrpc`, a subpackage of the root module)

`github.com/Bugs5382/go-apperr/apperrgrpc` is a plain subpackage: the repo has exactly one
`go.mod`, at the root, and one `vX.Y.Z` tag versions everything. The root module requires
`google.golang.org/grpc`, `google.golang.org/genproto/googleapis/rpc` and
`google.golang.org/protobuf` for it. A binary that imports only the root `apperr` package still
links no gRPC code, because Go's linker drops packages that are never imported.
`TestSingleModule` (`module_test.go`) fails if a nested `go.mod` ever appears again.

- `Code(Category) codes.Code`, `MetaCodeNum` (`"codeNum"`).
- `Status(ctx, reg, err, defaultCode, domain) *status.Status` and `Error(...) error`.
- `Info`, `FromStatus(*status.Status) (Info, bool)`, `FromError(error) (Info, bool)`.

## Layout

One module, `github.com/Bugs5382/go-apperr`, with a single `go.mod` at the root:

- `doc.go` - package overview.
- `apperr.go` - `Coded`, the internal `codedError`, `Code`.
- `sink.go` - the `Recorder`/`Logger` interfaces and their no-op defaults.
- `registry.go` - `Entry`, `Registry`, options, `NewRegistry`, and the present/describe/message/
  markdown methods.
- `category.go` - `Category`, its constants and `Registry.Category`.
- `fields.go` - request `Field`s for the sinks.
- `meta.go` - wire metadata: `WithMeta`, `Meta`, `Metadata`.
- `apperrgrpc/` - the gRPC adapter subpackage (see above).
- `module_test.go` - guard that the repo stays a single module.
- `*_test.go`, `example_test.go` - unit tests and verified `Example` functions.
- `examples/` - runnable `package main` programs: `coded`, `registry`, `markdown`, and `custom`
  (bring-your-own sinks over `log/slog`). They import only the core package and the standard
  library.

## Build, test, lint

- Build: `task build` (`go build ./...`)
- Test: `task test` (`go test ./...`); no external service/fixture required.
- Lint: `task lint` (gofmt check + `golangci-lint run` + `yamllint .`)
- Full local gate: `task ci` (build + `go vet` + test + lint)
- License headers: `task license` (check) / `task license:fix` (inject)

## Conventions and gotchas

- See `CLAUDE.md` for the branch/commit/PR rules; they are enforced by the git hooks in
  `.claude/hooks` (run `bash .claude/hooks/install.sh` once per clone).
- Open every PR as a draft. CI skips drafts, so run the full checks locally, push once they pass,
  and mark the PR ready when the work is finished; see CLAUDE.md "CI and Actions minutes".
- Keep the root `apperr` package standard-library only: it must import no logging, tracing, or
  third-party package. Observability backends are the consumer's own adapter, never added here. A
  transport adapter with dependencies goes in its own subpackage, like `apperrgrpc/`, inside the
  root module. Never add a nested `go.mod`: no Bugs5382 package ships nested modules, and
  `TestSingleModule` fails the build if one appears.
- `UserSafe` defaults to false and must stay that way: an entry is presented with only the
  template unless it opts in.
- Any change to `Recorder`, `Logger`, or an exported registry method is a public-API change: keep
  it additive unless a change is explicitly scoped as a major bump, and update this file and the
  README when the surface changes.
- `WithMessageTemplate` takes a fmt template that receives the code (include a `%d`); the default
  is `"Code %d: Internal Error"`.
