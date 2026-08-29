# Story Planning — Clarification Questions

Your answers are mostly unambiguous. One point needs resolution before generation.

## Ambiguity 1: Where do purely cross-cutting NFRs live? (Q6 = B)

You chose "fold NFRs into acceptance criteria of related functional stories only" (no dedicated NFR stories). Some NFRs map cleanly onto a functional story (e.g. NFR-1 performance → the pricing story; NFR-3 secret hygiene → the output stories; NFR-4 → the credentials story). But three have no natural single home:

- **NFR-2.3** — Property-based testing setup (framework `rapid`, generators, CI seed logging)
- **NFR-5** — Maintainability/extensibility (the "add a resource type via one interface" design contract)
- **NFR-6** — Portability (single static binary for 5 OS/arch targets)

### Clarification Question 1
How should these three be represented?

A) Attach them as acceptance criteria to the most-related functional story anyway (PBT → JSON/usage round-trip story; extensibility → pricing-catalog story; portability → the "install and run" story)

B) Allow a small "Quality & Platform" group with a dedicated story for each of these three only, keeping everything else folded in per Q6

C) Leave them out of stories.md entirely and carry them only via the requirements → NFR-design/build stages

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Note (no answer needed): ID scheme interpretation

Q1 = User Journey-Based grouping, Q7 = `EPIC-<area>` + `US-<area>-<n>`. I will treat each journey/epic as the `<area>` (e.g. `EPIC-LOCAL-ESTIMATE` with stories `US-LOCAL-ESTIMATE-1`, `US-LOCAL-ESTIMATE-2`, …). Tell me if you'd prefer shorter area slugs.
