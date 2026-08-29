# Personas — Price Checker

These personas drive CLI ergonomics, output-format design, and the v1 scope cut. The CI Pipeline persona is a non-human (automated) consumer.

---

## P1 — Dana, the Local Developer

- **Role**: Application / infrastructure developer who writes Terraform for their team's services.
- **Context**: Works on a laptop (macOS or Linux). Has AWS credentials configured via a named profile or SSO. Runs `terraform plan` frequently while iterating on a change.
- **Goals**:
  - See roughly what a change will cost before opening a pull request.
  - Understand which resource in the plan is driving a cost jump.
  - Not have to learn a new config language or sign up for a service.
- **Frustrations**:
  - Cloud pricing pages are unreadable; mental math is error-prone.
  - Tools that need a server, an account, or an API token just to price a plan.
  - Slow tools that re-download pricing data every run.
- **How they use the tool**: `price-checker --path .` or pipes `terraform show -json` into it; occasionally `--format json | jq`. Reads the table, expands components with `-v`.
- **Key needs**: Fast warm runs, sensible defaults, a readable table, a clear "not estimated" list so numbers aren't silently wrong.

---

## P2 — Sam, the Platform / DevOps Engineer

- **Role**: Owns the shared Terraform modules, the CI pipeline, and the team's tooling standards.
- **Context**: Sets up `price-checker` once for the whole team — a committed `price-checker.yml`, a shared `infra-usage.yml`, and the CI job. Cares about reproducibility and supply-chain hygiene.
- **Goals**:
  - Pin defaults (region, cache TTL, thresholds, usage file) so every developer and the CI job behave identically.
  - Keep the estimate stable across runs and machines.
  - Extend coverage when the team adopts a new AWS service.
- **Frustrations**:
  - Per-developer flag drift producing inconsistent numbers.
  - Tools that can't be scripted cleanly (mixed stdout/stderr, unstable output).
  - Having to fork a tool to add one resource type.
- **How they use the tool**: Authors config and usage files; wires `--format github-comment` and `--threshold-monthly` into CI; runs `--log-level debug` when coverage looks wrong.
- **Key needs**: Config file with clear precedence, stable JSON schema, documented exit codes, an extension interface for new resource types, single static binary to vendor.

---

## P3 — The CI Pipeline (automated consumer)

- **Role**: Non-human. A GitHub Actions / GitLab CI / Jenkins job that runs `price-checker` on every pull request.
- **Context**: Ephemeral runner with an OIDC-assumed AWS role (read-only pricing) and the `terraform` binary available. No TTY. Network may be slow or rate-limited.
- **Goals**:
  - Produce a cost breakdown and a cost diff for the PR.
  - Fail the build when a change exceeds an agreed cost budget.
  - Emit a comment body for the pipeline to post.
- **Frustrations**:
  - Non-deterministic output that breaks diffing or snapshotting.
  - Ambiguous exit codes (can't tell "tool broke" from "cost too high").
  - Colour codes and progress spinners polluting logs.
- **How it uses the tool**: `price-checker diff --compare-to <base-plan> --format json`, then `--format github-comment --out comment.md`; branches on exit code.
- **Key needs**: Deterministic output, `NO_COLOR`/non-TTY behaviour, distinct exit codes (runtime error / strict failure / threshold breach), stdout-only data with logs on stderr.

---

## P4 — Priya, the PR Reviewer / Approver

- **Role**: Senior engineer or tech lead who reviews infrastructure pull requests. Does **not** run the tool herself.
- **Context**: Reads the cost comment the CI pipeline posts on the PR. Decides whether a cost increase is acceptable.
- **Goals**:
  - See the monthly cost delta of the change at a glance.
  - Know which resources changed and by how much.
  - Trust the number, or clearly see what wasn't estimated.
- **Frustrations**:
  - A wall of numbers with no summary.
  - A total that looks precise but silently omitted half the resources.
  - Comments that get re-posted on every push, burying the discussion.
- **How they use the tool**: Reads the rendered Markdown comment and, occasionally, the HTML report artifact.
- **Key needs**: A one-line headline delta, a compact changed-resources table, an explicit not-estimated count, collapsible detail.

---

## P5 — Alex, the FinOps / Cost Owner

- **Role**: Owns the cloud bill across teams. Sets cost policy.
- **Context**: Doesn't write Terraform but defines the org-wide usage assumptions and the cost thresholds that gate pipelines. Reviews HTML reports and JSON output in aggregate.
- **Goals**:
  - Standardize monthly usage assumptions (S3 volume, Lambda invocations, NAT data processed) across teams.
  - Set and adjust per-repo or per-environment cost budgets.
  - Get consistent, exportable figures for tracking.
- **Frustrations**:
  - Every team guessing different usage numbers.
  - No machine-readable output to feed into their own dashboards.
  - Estimates that quietly ignore usage-based costs entirely.
- **How they use the tool**: Maintains the shared usage YAML; sets `--threshold-monthly` values; consumes `--format json` and HTML reports.
- **Key needs**: A documented usage-file schema, a generator to discover which components need assumptions, stable JSON with totals and per-resource figures, USD monthly figures.

---

## Persona Priorities at a Glance

| Concern | P1 Dana | P2 Sam | P3 CI | P4 Priya | P5 Alex |
|---|---|---|---|---|---|
| Fast local table | High | Med | — | — | — |
| Config file & defaults | Low | High | Med | — | Med |
| Stable JSON schema | Med | High | High | — | High |
| Exit codes & thresholds | Low | High | High | Low | High |
| PR comment / diff | Med | High | High | High | Med |
| Usage file & schema | Low | High | Med | — | High |
| Not-estimated transparency | High | High | Med | High | High |
| Extensibility (new resource types) | Low | High | — | — | Low |
