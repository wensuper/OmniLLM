# Security policy

English | [简体中文](docs/zh-CN/security-policy.md)

## Reporting a vulnerability

**Please don’t report security problems in public issues, discussions, or pull requests.**

Report them privately through GitHub’s **Security → Report a vulnerability**:
<https://github.com/wensuper/OmniLLM/security/advisories/new>

Please include, as far as you can:

- the affected component (hub, device, connection service, web client, or installer) and version (`omnillm version`);
- steps to reproduce, or a proof of concept;
- the impact as you see it: who can do what.

We reply within 3 business days. Once a report is confirmed, we fix it as soon as we can, publish a new release, and credit you in the changelog (unless you’d rather not be named). Please don’t disclose details until the fix is released.

## Scope

We particularly care about:

- the connection service (or anyone on the network) reading or changing conversations, or impersonating a hub;
- pairing a device without the hub owner’s confirmation;
- credentials (API keys, subscription sign-ins) leaving the hub, or appearing in logs, the user interface, or data sent to devices;
- a device making Claude Code or Codex on the hub run commands, read or write files, or read the owner’s personal settings;
- background programs reaching subscription models, or getting around rate limits;
- other users on the same computer, or a malicious web page, abusing the local console or local API;
- the installer or the release process being made to install a different program.

Known limitations described in [Security](docs/en/security.md) — for example, that the connection service can see metadata, or that the first load of the web client trusts its host — are not vulnerabilities. Ideas for improving them are still welcome.

## Supported versions

OmniLLM is in pre-release; only the latest release receives fixes. Please update to the latest version (`omnillm update`) and check that the problem is still there before reporting it.
