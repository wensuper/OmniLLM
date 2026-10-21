# Contributing to OmniLLM

English | [简体中文](docs/zh-CN/CONTRIBUTING.md)

Thanks for your interest in OmniLLM! Issues and pull requests are welcome. This guide explains how to get a change merged.

- [Before you start](#before-you-start)
- [Development setup](#development-setup)
- [Making a change](#making-a-change)
- [Commit messages](#commit-messages)
- [Pull requests](#pull-requests)
- [Reporting bugs and requesting features](#reporting-bugs-and-requesting-features)
- [License](#license)

## Before you start

Please read these first. The design documents are in Chinese for now; a machine translation is fine for reading them.

1. **[Principles](docs/principles.md)** (in Chinese). A change that breaks any of them won’t be merged — above all the first two: only the hub talks to model providers, and the hub and its devices talk only through the connection service, end-to-end encrypted.
2. **[Design](docs/design.md)** (in Chinese). If your change alters behaviour, update it in the same pull request.
3. **[Security](docs/en/security.md)**. These are promises to users; a change must not make any of them untrue.

For anything larger than a small fix, please open an issue first so we can agree on the approach before you spend time on it.

Security problems are reported privately: see [SECURITY.md](SECURITY.md). Don’t open a public issue for them.

## Development setup

You need Go (see `go.mod` for the version).

```bash
go build ./...
go test -race ./...
go vet ./...
scripts/build-web.sh          # build the web client (WebAssembly) into dist/web
```

Don’t run automated tests against the hub you actually use: pairing and removing devices affects the devices you already have. Start a separate hub with its own data folder:

```bash
OMNILLM_HOME=/tmp/omnillm-dev OMNILLM_RELAY=off go run ./cmd/omnillm hub run
```

## Making a change

- **Fix bugs test-first.** Add a test that reproduces the bug, then fix it.
- **Security rules live in code**, guarded by tests. A rule that exists only in documentation or in a prompt doesn’t count.
- **Words people read** — error messages, the console, the web client, the CLI — ship in both English and Chinese. Say what happened and what to do next. Never show protocol fields, error codes, or internal names: say “connection service”, not `relay`; “encrypted”, not `Noise`.
- **Dependencies.** Explain why a new dependency is needed. OmniLLM is first a Go library that other programs embed, so the library itself reads no configuration, opens no ports, writes nothing to standard output, and keeps no global state.
- **Documentation.** User documentation exists in English (`docs/en/`) and Chinese (`docs/zh-CN/`). If you change one, change the other in the same pull request. A release adds an entry to both changelogs: [CHANGELOG.md](CHANGELOG.md) and [docs/zh-CN/CHANGELOG.md](docs/zh-CN/CHANGELOG.md).

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/): start the subject with `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `perf:`, `ci:`, or `chore:`, and describe the change from the user’s point of view where you can.

```text
fix: closing the tab in use lets the other tab take over by itself
```

## Pull requests

Before you open a pull request:

- [ ] `go test -race ./...`, `go vet ./...` and `staticcheck ./...` pass.
- [ ] New behaviour has tests; bug fixes have a test that failed before the fix.
- [ ] User-facing text is in English and Chinese.
- [ ] `docs/design.md` and the user docs in both languages are updated if behaviour changed.

CI runs the same checks on macOS and Linux. A maintainer reviews every pull request; please keep each one focused on a single change.

## Reporting bugs and requesting features

Use the [issue templates](https://github.com/wensuper/OmniLLM/issues/new/choose). For a bug, include the output of `omnillm version` and `omnillm status`, what you did, what you expected, and what happened instead. Remove API keys and personal details from logs before you paste them.

## License

By contributing, you agree that your contributions are licensed under the [Apache License 2.0](LICENSE).
