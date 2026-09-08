# Project instructions

## Writing style

- Never use an em dash (—) in any file: docs, code comments, commit messages. Use a comma, colon, semicolon, parentheses, or a period instead.

## Documentation

- All specs and decisions live in `KB/`, starting from `KB/0001-purpose.md`. See `KB/README.md` for naming and cross-referencing conventions before adding a new document.
- When an implementation change alters behavior a `KB/` doc describes (a new command, flag, menu option, field, or rule; a changed default; a removed capability), update that doc in the same change so the spec matches what's actually built. Don't let drift accumulate for someone to reconcile later.

## Go

- Go modules live under directories such as `cli/` (module `endonend/cli`); see `KB/0002-architecture.md` for which components are Go. Match the Go version pinned in the relevant `go.mod`.
- Every change to a `.go` file must come with tests in the same package (`_test.go`, table-driven where it fits). Run `go test ./...` from the module root and confirm it passes before considering the change done.
- Lint with `golangci-lint run ./...` from the module root, against the shared `.golangci.yml` at the repo root. Fix findings instead of suppressing them; a `//nolint` must carry a comment explaining why.
- Git hooks in `scripts/git-hooks/` run `go test ./...` and `golangci-lint run ./...` for every Go module on each commit and push, and block the commit or push on failure. One-time setup per clone: `git config core.hooksPath scripts/git-hooks`.
- `.github/workflows/go.yml` runs the same build, vet, test, and lint steps in CI on every push to `main` and every pull request, per `KB/0002-architecture.md`'s CI/CD section. Add a new `matrix.module` entry there whenever another Go module is added (e.g. the endonend platform backend).

### Go best practices

- Run `gofmt` (or `goimports`) on every file before committing; formatting is not a matter of taste here.
- Return errors, don't panic, except for truly unrecoverable startup failures (e.g. a malformed config read once at boot). Wrap errors with context using `fmt.Errorf("...: %w", err)` so callers can `errors.Is`/`errors.As` against the original.
- Check every returned error. Don't discard one with `_` unless it is genuinely safe to ignore (e.g. a `Close()` on a read-only file), and say why in a comment when it isn't obvious.
- Keep package names short, lowercase, and free of underscores or stutter (`signing.Sign`, not `signing.SigningSign`). A package's exported surface should be small; keep implementation details unexported.
- Prefer accepting interfaces and returning concrete types. Don't define an interface until there are two real implementations or a test needs to substitute one.
- Thread `context.Context` as the first parameter through any call that does I/O (network fetches, file reads over a deadline, crawler/validator calls per `KB/0002-architecture.md`), and respect cancellation.
- No global mutable state. Pass dependencies explicitly (constructor params or small structs) instead of package-level variables, so tests can run in isolation and in parallel.
- Table-driven tests are the default for anything with more than one interesting case; name subtests with `t.Run` so failures are easy to locate.
- Keep functions short and single-purpose; extract a helper when a function mixes more than one level of abstraction, but don't pre-emptively abstract for hypothetical future cases.

## React

Web frontend code lives under `web/` per `KB/0002-architecture.md`: TypeScript and React (Next.js), server-rendered for the endonend platform's discovery site and statically exported per artist/label as a self-hostable storefront from the same codebase. Keep this dual-mode constraint in mind before adding any server-only or client-only API to shared components.

- Function components with hooks only; no class components.
- Type props and state explicitly; avoid `any`. Derive types from the manifest JSON Schema (`KB/0003-manifest.md`) rather than hand-duplicating shapes that can drift from it.
- Keep components small and single-purpose. Lift state up only as far as it needs to go; don't reach for Context or an external store until prop drilling actually causes pain.
- Keep side effects (fetching, subscriptions, timers) out of render, in `useEffect` or a server component/loader, and clean them up in the effect's cleanup function.
- Prefer server components and server-side data fetching where Next.js supports it; reserve client components for actual interactivity.
- Don't reach for `useMemo`, `useCallback`, or `React.memo` preemptively; add them only in response to a measured re-render problem, not by default.
- Co-locate a component's tests with the component file rather than in a parallel test tree.

### React / web testing

- Unit and component tests: Vitest (or Jest) with React Testing Library, asserting on rendered output and behavior a user would see, not internal state or implementation details.
- Every new or changed component and hook needs a test alongside it, mirroring the Go testing rule above; run the test suite and confirm it passes before considering a change done.
- Test both rendering modes a component can run in when it's shared: at least one check against the server-rendered discovery site and one against the statically-exported storefront output, since a bug can pass in one mode and fail in the other.
- Before reporting any UI or frontend change as complete, start the dev server and exercise the feature yourself in an actual browser, golden path and edge cases, not just green tests; type checking and unit tests verify code correctness, not that the feature actually works.
- Once the frontend exists, script the same browser checks with Playwright for repeatable end-to-end coverage; until then, manual browser verification is the minimum bar, not optional.
