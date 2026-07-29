# go-hexa — a modular, functional, multi-frontend Go starter

A **reusable Go foundation** under construction: modular hexagonal architecture, functional
programming, exposed to **N simultaneous frontends** (web, mobile, CLI, events).

This is not an application, and not a project template you copy once either. The target is a
**framework**: a core of reusable modules, a generator (`hexa new`), and a *tooled* rulebook that
stops the shape from degrading. The path there is described in
[`documentation/technique/parite-frameworks.md`](documentation/technique/parite-frameworks.md).

> **Licensed under [Apache-2.0](LICENSE)** since 2026-07-29 — use, modification and redistribution
> are granted, with an explicit patent grant. The decision and what it makes irreversible are
> recorded in [ADR 020](documentation/adr/020-la-licence-est-apache-2-0.md).
>
> **⚠️ Open licence, but not yet importable.** Every package still lives under `internal/`, which
> the Go compiler forbids third parties from importing. `go list ./... | grep -v /internal/` returns
> five packages, and **all five are `package main`** — so the number a third party can import is
> **zero**. The licence removed the legal blocker; the public frontier is a separate piece of work,
> and [ADR 015](documentation/adr/015-la-frontiere-publique-est-derivee-d-un-usage-mesure.md)
> requires it to be derived from measured usage rather than declared. Until then, `hexa new` copies
> the starter — it is not a dependency you can `require`.

> **State of play.** The HTTP server, the command line and the event dispatcher exist and run — the
> full chain `registration → outbox → dispatcher → relay → idempotency guard → notification` has
> been exercised on the real binaries. **`auth` and `notification` now exist**; what is missing is
> **multi-tenancy**, **payment** and **rate limiting**, described with no code at all.
>
> The factual record — dated, and without indulgence — is in
> [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md) § *État réel du dépôt*. **It, not this
> README, is authoritative on facts.** The known gaps between what is written and what is are listed
> in [`documentation/process/AUDIT_CONFORMITE.md`](documentation/process/AUDIT_CONFORMITE.md).

## In thirty seconds, with nothing installed

```bash
export SECURITY_ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)
go run ./cmd/server
```

```bash
curl -s -X POST localhost:8080/v1/users \
  -H 'content-type: application/json' \
  -d '{"email":"Alice@Example.COM ","password":"correct horse battery staple"}'
```

```json
{
  "user_id": "019f9b46-3aec-735a-977d-129192ef130f",
  "email": "alice@example.com",
  "status": "pending",
  "created_at": "2026-07-25T21:54:58.924Z"
}
```

No database, no Redis, no Docker. Three things are readable in that reply: the identifier is a
**UUID v7** — time-ordered, therefore usable as a primary key without fragmenting the index; the
address is **normalised by the domain**, spaces and case included; and the account is born
**`pending`**, never active, because the address has not been proven yet.

Interactive documentation is on `/docs`, the contract on `/openapi.json` and `/openapi.yaml`.

## Two properties, and everything else follows

**1. The core knows nothing.** Not HTTP, not SQL, not a cache, not a clock, not a logger. It
receives functions, it returns values. It is tested in microseconds, without a container.

**2. The number of frontends is a non-subject.** A surface is a primary adapter plugged into the
same use cases. Adding one changes no file of the core.

## The vocabulary, in four words

Their meaning is fixed by
[ADR 012](documentation/adr/012-anatomie-d-un-module-et-pilotes.md),
[ADR 019](documentation/adr/019-l-anatomie-nomme-ses-adaptateurs.md) and
[`rules/references.md`](rules/references.md). The word **service** is proscribed: it means three
different things depending on who says it.

| Word | Meaning | Where |
|---|---|---|
| **core module** | reusable technical capability, shipped by the starter | `internal/core/{name}/` |
| **business module** | a bounded context of an application | `internal/modules/{name}/` |
| **driver** | one interchangeable implementation of a module | `.../drivers/{name}/` |
| **surface** | a served frontend — web, mobile, CLI, events | `.../adapters/primary/{name}/` |

Both kinds of module have **the same anatomy**. A business module knows no other business module; it
consumes the core's ports.

## Zero infrastructure prerequisite

Every module has a driver with **no external dependency**, chosen by default:

| Core module | Default | Also available |
|---|---|---|
| `outbox` | `memory` | `postgres` |
| `idempotency` | `memory` | `postgres` · `redis` |
| `dynconf` | `file` | `postgres` |
| `audit` | `log` | `postgres` |
| `storage` | `disk` | — |
| `scheduler` | `cron-inproc` | `advisory-lock` |
| `auth` | `memory` | — |
| `notification` | `log` | — |

The rule holds for **business** modules too: `user_registration` has its `memory` driver, and that
one is the default. A business module whose only driver required PostgreSQL would break the promise
at the very first module written — that is, exactly when it gets tested.

With this configuration nothing is required: no database, no Redis, no Docker. A test locks the
promise down — every active module, every one on its default driver, must require no service.

**Every driver documents its NON-guarantees** at the top of its package. An in-memory driver does not
survive a restart and shares nothing between replicas: it is written, in capitals, right where it
gets wired.

The promise covers **infrastructure**, not secrets. One variable stays mandatory,
`SECURITY_ENCRYPTION_KEY`, and it will never have a default value — a default encryption key would
encrypt everyone's data with a publicly known one.

## Getting started

```bash
git config core.hooksPath .githooks   # guard rail against pushing straight to the trunk
task init                             # .env + tooling
task check                            # fmt · vet · lint · arch · test · vuln
```

**Docker is not required** to develop: `go test ./...` without a tag needs no service. The levels
that do (`-tags=integration`, `-tags=e2e`) are provided by CI.

Three binaries:

| Command | Role | Prerequisite |
|---|---|---|
| `go run ./cmd/server` | HTTP surfaces | none |
| `go run ./cmd/cli` | `register` · `seed` · `health` — the **same port** as HTTP | none |
| `go run ./cmd/worker` | dispatching the outbox towards the broker | an outbox **shared across processes** |

> ⚠️ The dispatcher **refuses to start** on the `outbox: memory` driver, and that is intended: that
> driver lives inside the process, so a separately launched worker would dispatch its own store —
> empty — while the server's events stayed in the server's memory. It would run publishing nothing
> **and reporting no error at all**. A silently inert component is the only defect that never
> announces itself.

## Starting your own project from this one

```bash
task rename -- github.com/{org}/{project}
hexa new ./my-project --module github.com/{org}/{project} --from .
hexa make:feature order_tracking --into ./my-project
```

The module path is the **only** naming value in the repository. No handle, no team, no `CODEOWNERS`:
the constraints bear on **rules** checked by CI, not on people. The starter works with one
contributor as well as with twenty.

`hexa make:feature` writes the whole anatomy — domain, ports, use case, dependency-free driver,
catalogue, local composition root, **HTTP surface** and tests at four levels — then **exercises the
entire project** before handing back. It also writes the new module's `arch-go` sealing rule: a
module no rule guards is indistinguishable from a guarded one.

**Measured on 2026-07-29**: clone → two live business modules, `task check` green inside the
generated project, in **6 seconds**. See
[`documentation/produit/preuves/`](documentation/produit/preuves/).

`user_registration` is the **reference slice**, not the application. It exists to show the complete
shape — pure domain, ports as function types, composed pipeline, interchangeable drivers, adapters
per surface — because that shape is what will be copied to write `billing` or `crm`. **Any folder
missing from it would be reproduced as "not necessary."**

## Layout

```
rules/                        engineering rulebook — AUTHORITATIVE
documentation/adr/            architecture decisions — AUTHORITATIVE
documentation/AMORCAGE.md     bootstrap and factual record — read this first
documentation/produit/        personas, scope, per-version matrix, per-persona proofs
config/*.yaml                 configuration by group, secrets through ${VAR} only
cmd/{server,worker,cli}       composition roots — the only code allowed to know everything
cmd/hexa                      the generator — a shell over internal/generator (ADR 016)
internal/pkg/                 dependency-free primitives: result · fp · pagination · middleware
internal/infrastructure/      technical foundation without business: db · cache · http · telemetry
internal/contracts/           published language: what modules exchange without importing each other
internal/core/{name}/         CORE MODULE — shipped by the starter
internal/modules/{name}/      BUSINESS MODULE — written by the application
  ├── domain/                 PURE — value objects, rules, errors, events
  ├── ports/                  function types ONLY
  ├── application/            use-case pipeline + decorators, no I/O
  ├── drivers/{name}/         one interchangeable implementation of the module
  ├── adapters/primary/       http · cli · events — one surface per folder
  ├── adapters/secondary/     hashing · outboxpub · postgres · mailer
  ├── tests/                  black box, one file per test
  ├── catalog.go              DECLARABLE drivers — shares its constants with New (ADR 014)
  └── module.go               local composition root — the ONLY file that knows the drivers
migrations/{engine}/          versioned SQL, backward compatible N-1 · `postgres/` only, so far
deploy/postgres/              provision.sql — the ROLES, run once, outside goose
deploy/toolbox/               the tooling AS AN IMAGE — nothing installed on the machine
tests/{e2e,integration,perf}  build tags — outside the default `go test ./...`
tools/*.sh                    the guards, ONE definition called by CI and by `task`
```

## What makes the framing hold

**A rule that is not tooled does not exist.** Every constraint has its guard — and the *proven*
column says whether that guard has **already run** on this repository, because a guard that never
ran guards nothing.

| Constraint | Guard | Proven |
|---|---|---|
| The core imports neither transport, nor persistence, nor a logger | `arch-go` · `depguard` | **yes** |
| A business module does not import another business module | `arch-go` | **yes** |
| A core module knows no business module | `arch-go` | **yes** |
| A port is a function type, not an interface | `arch-go` | **yes** |
| A binary exports nothing but `main` | `arch-go` | **yes** |
| Short functions, few parameters, bounded complexity | `funlen` · `cyclop` · `arch-go` | **yes** |
| No ignored error, no non-exhaustive `switch` | `errcheck` · `exhaustive` | **yes** |
| No global state, no `func init()` | `gochecknoglobals` · `gochecknoinits` | **yes** |
| No ORM | `depguard` | **yes** |
| The shipped configuration loads and requires no service | `internal/config/tests/` | **yes** |
| No commit straight to the trunk | server ruleset + `pre-push` hook | **yes** — proven by a real `GH013` refusal |
| Coverage ≥ 70 % unit scope, ≥ 90 % core | `tools/covergate`, ratchets | **yes** — 79.4 % · 92.4 % · 62.8 % |
| The code language is English | `tools/verifie-langue-du-code.sh` | **yes** (ADR 018) |
| No debt hidden as a code marker | `tools/verifie-dette.sh` | **yes** |
| The documentation tells the truth about what can be checked | `tools/verifie-veracite-doc.sh` | **yes** (#118) |
| A module does not reach another one's SQL schema | `deploy/postgres/verify.sql` | CI only |
| The audit log refuses `UPDATE` and `DELETE` | CI job `migrations` | CI only |
| A migration's rollback works | CI job `migrations` (it **replays** it) | CI only |
| No secret in the history | `gitleaks` | **yes** — 32 commits, no leak |
| Touching the rulebook requires an ADR | CI job `inertia` | **yes** |
| No known vulnerability | `govulncheck` · CodeQL | **yes** — 0 and 0 |

**Measured on 2026-07-29**: `golangci-lint` returns **0 findings** (~50 analysers, down from 239),
`arch-go` **21 rules out of 21** with **100 % coverage** — 14 of them dependency rules —, and
`govulncheck` **0 vulnerabilities**. All of them actually ran, inside the `task check` chain, and
**the exit code was checked** — not just the output.

### Every guard ships with the case that makes it fail

That is [ADR 013](documentation/adr/013-un-garde-doit-savoir-echouer.md), and it was written because
**eleven guards of this repository guarded nothing**. The shape is always the same: a guard that
finds nothing looks exactly like a satisfied guard.

Each `tools/verifie-*.sh` therefore carries a `--temoin` mode proving it still refuses **and** that
it can still be satisfied. Without the second half, a broken guard and a strict guard are
indistinguishable — a lesson learned from a guard that passed its refusal case *because its control
function did not exist*.

## Stack

| Layer | Choice | Why |
|---|---|---|
| Routing | `chi` | 100 % `http.Handler` — reversible in a day ([ADR 008](documentation/adr/008-chi-huma-plutot-qu-un-framework.md)) |
| Contract | `huma` v2, code-first | Served on `/openapi.{json,yaml}`; client SDKs derive from it |
| Persistence | `pgx` v5, explicit SQL | No ORM: they leak into the domain ([ADR 009](documentation/adr/009-strategie-d-acces-aux-donnees.md)) |
| Database engine | **none imposed** | `postgres` is one driver among others (issue #36) |
| Asynchronous | transactional outbox | Neither loss nor phantom ([ADR 006](documentation/adr/006-outbox-transactionnel.md)) |
| Wiring | manual composition | Checked by the compiler ([ADR 004](documentation/adr/004-composition-manuelle-sans-conteneur-di.md)) |
| Observability | OpenTelemetry + `slog` | Traces, metrics and logs tied by `trace_id` |

## The documentation never lies about the real state

That is a golden rule, not an intention. Three levels, never conflated:

- **proven locally** — the command ran on the reference machine and its **exit code** was checked;
- **written, not proven** — the code exists and compiles; nothing has run it;
- **never deployed** — `deploy-uat.yml` and `deploy-production.yml` have never run.

A document that ticks "✅ tested" without a test is worse than no document. The complete record, with
its date, is in [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md).

Three caveats worth keeping, all of them measured:

- **`auth` exists, and that does not mean "no vulnerabilities."** The module is new, it has never
  been exercised anywhere but here, and nothing has been audited by a third party.
  `GET /v1/users/availability` still allows **enumerating** registered addresses — acceptable behind
  rate limiting, and the `ratelimit` module **does not exist**.
- **The event consumer works, but is not mounted as a surface.** It is wired inside `cmd/worker`
  rather than in `adapters/primary/events/`: the third primary adapter does not exist yet
  ([#9](https://github.com/SteelHeart/go-hexa-fp-starter/issues/9)). And `notification` only has a
  `log` driver — **no email has been sent anywhere**
  ([#27](https://github.com/SteelHeart/go-hexa-fp-starter/issues/27)).
- **This repository cannot yet be depended upon.** Measured for persona P3: the five packages outside
  `internal/` are all `package main`, so the number of packages a third party can actually import is
  **zero**. See
  [`documentation/produit/preuves/p3-adoption-externe.md`](documentation/produit/preuves/p3-adoption-externe.md).

## A note on language

The **code** is in English — identifiers, godoc, internal error messages, tests
([ADR 018](documentation/adr/018-la-langue-du-code-est-l-anglais.md)). The **rulebook**, the ADRs and
the process documents are in **French**: they are the working language, and they are not published
with the code.

The dividing line is *what ships with the code*. This README ships with it, so it is in English.

⚠️ Messages meant for the **end user** — the ones that leave in a `422` response body — are still in
French. They are product content, not code language, and they are waiting for internationalisation
([#12](https://github.com/SteelHeart/go-hexa-fp-starter/issues/12)). That is written down so nobody
translates them "along the way" and makes it look like the question is settled.

## Documentation

| I am looking for | It is here |
|---|---|
| Where to start, and the real state | [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md) |
| The rulebook | [`rules/README.md`](rules/README.md) |
| What is forbidden | [`rules/interdictions.md`](rules/interdictions.md) |
| The bar to ship | [`rules/definition-of-done.md`](rules/definition-of-done.md) |
| Why a given decision | [`documentation/adr/`](documentation/adr/README.md) |
| A module's anatomy | [ADR 012](documentation/adr/012-anatomie-d-un-module-et-pilotes.md) · [ADR 019](documentation/adr/019-l-anatomie-nomme-ses-adaptateurs.md) |
| Who this is for, and the proofs | [`documentation/produit/personas.md`](documentation/produit/personas.md) · [`preuves/`](documentation/produit/preuves/) |
| The planned core modules | [`documentation/technique/modules-noyau.md`](documentation/technique/modules-noyau.md) |
| The driver catalogue | [`documentation/technique/pilotes.md`](documentation/technique/pilotes.md) |
| Known gaps between written and real | [`documentation/process/AUDIT_CONFORMITE.md`](documentation/process/AUDIT_CONFORMITE.md) |
| Naming a branch, a commit, a file | [`documentation/process/NOMENCLATURE.md`](documentation/process/NOMENCLATURE.md) |
| Contributing | [`rules/workflow-git.md`](rules/workflow-git.md) |
| Reporting a vulnerability | [`SECURITY.md`](SECURITY.md) |

## Licence

[`LICENSE`](LICENSE) — **Apache License 2.0**, since 2026-07-29. Use, modification and
redistribution are granted, along with an **explicit patent grant** (§3) — the reason Apache-2.0 was
picked over MIT for a starter meant to be embedded in third-party products.

Attribution notices are in [`NOTICE`](NOTICE). Contributions are covered by §5: anything you submit
for inclusion is under these same terms, so there is no CLA to sign.

The decision is recorded in [ADR 020](documentation/adr/020-la-licence-est-apache-2-0.md). It
reverses an earlier ruling — the repository was deliberately *source-available* (all rights
reserved) until the persona proofs showed that stance was incompatible with the transfer condition:
an external adopter could read this code and legally do nothing with it.

> ⚠️ **What the licence does not do.** It does not make this importable. Every package is still
> under `internal/`; the public frontier has to be built, and ADR 015 requires it to be derived from
> measured usage rather than declared up front. Publishing a package under this licence cannot be
> undone — which is exactly why it is not being rushed.
