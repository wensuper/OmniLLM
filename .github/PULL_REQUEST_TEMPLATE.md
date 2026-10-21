## What and why

<!-- What does this change, and why? Link the issue it addresses (Fixes #123). -->

## How it was tested

<!-- Tests added or changed; anything you checked by hand (console, phone, menu bar). -->

## Checklist

- [ ] `go test -race ./...`, `go vet ./...` and `staticcheck ./...` pass
- [ ] Bug fixes include a test that failed before the fix
- [ ] User-facing text is in English and Chinese
- [ ] Docs updated in both `docs/en/` and `docs/zh-CN/` (and `docs/design.md` if behaviour changed)
- [ ] Doesn’t break the [principles](../docs/principles.md): only the hub talks to model providers; hub and devices talk only end-to-end encrypted through the connection service
