# Application Design — Clarification Questions

Almost all answers are consistent. One creates a conflict with approved requirements.

---

## Contradiction 1: Pricing layers (Q6 = B) vs caching / dedup / concurrency requirements

Q6 = B says *"each `Pricer` talks to the AWS SDK directly and does its own caching."*

That conflicts with several approved requirements and stories:

- **US-PRICING-6 / NFR-1.2** — "Price List queries for a run are **deduplicated** and issued **concurrently with a bounded worker pool**." A shared dedup + worker pool is hard if every pricer calls the SDK independently.
- **US-CACHE-1/2/3 / FR-4** — one on-disk cache with a TTL, `--no-cache`, `--refresh-cache`, atomic writes, concurrency-safety, and a **cache hit ratio + query count** in JSON metadata (NFR-7.2). That's naturally one cache component, not N per-pricer caches.
- **US-PRICING-7 / NFR-5.1** — "add a resource type by implementing **one interface**." If that interface also owns SDK calls and caching, every new pricer re-implements (and must be tested against) SDK mocking and cache handling.

Options A and C keep the pricer contract small and put the AWS SDK + cache + dedup + worker pool behind a shared boundary.

### Clarification Question 1
How should the pricing side be layered?

A) **Two layers.** `PriceListClient` = thin shared wrapper over the AWS SDK `GetProducts` (owns credentials, region routing, retry/backoff, the bounded worker pool, and per-run dedup). The on-disk cache is a decorator wrapping this client. A `Pricer` per resource type only builds queries, picks price dimensions, and emits cost components — it receives the client as a dependency and never imports the AWS SDK or touches the cache.

B) **Three layers.** As A, plus a `PriceSet` built once per run that pre-resolves every price the plan needs into memory; `Pricer`s become pure functions over that in-memory `PriceSet` (best for property-based testing and determinism, slightly more upfront wiring).

C) Keep Q6 = B (each pricer calls the SDK and caches itself) and I accept that dedup, the shared worker pool, unified cache stats, and the thin pricer contract are downgraded / handled ad hoc.

X) Other (please describe after [Answer]: tag below)

[Answer]: A  (provided in chat 2026-08-29)

---

## Notes (no answer needed)

- **Q4 = `math/big.Rat`**: workable and exact. AWS price strings parse losslessly via `big.Rat.SetString`. The rounding/formatting policy (half-up to cents for display, full precision internally) will be pinned in `U3 cost-engine` Functional Design.
- **Q9 = `html/template` + inlined JS charting lib**: acceptable with FR-10 (everything `//go:embed`-inlined, zero external requests). To keep the binary lean I'll propose a single small library (e.g. a ~10 KB minified sparkline/bar helper) rather than a full charting framework; you can veto the specific lib at code-generation planning.
- **Q10 = authored JSON Schema, embedded + validated in tests**: Go structs still exist for marshalling; the committed schema file is the consumer-facing contract and tests assert output conforms.
