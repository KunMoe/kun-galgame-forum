<!--
  /api/v1 rebuild PRs: the SOP is refs/docs/proj/api-v1/05-session-sop.md
  (gitignored: read it in the source checkout, not the worktree).
  Anything else: delete the whole template and describe the change.
-->

## What

<!-- Which domain moves onto /api/v1, which legacy routes go, which migration numbers. -->

Wave:
Contract: `refs/docs/proj/api-v1/waves/`
Legacy routes removed: N (`legacy_route_baseline` A → B)
Migrations: none / NNN–NNN

## Gates

Nine gates, run locally. CI covers 1, 3, 4 and 8 only — the rest have no automation.

- [ ] 1 · `GOTOOLCHAIN=go1.26.1 make lint` + `go vet` — no output
- [ ] 2 · contract gates G2–G17 / F1–F10
- [ ] 3 · spec consistency, no generated-file drift
- [ ] 4 · DB tests on a throwaway database, `-count=1 -p 1`, `KUN_REQUIRE_TEST_DB=1`
- [ ] 5 · full cursor traversal matches a direct SQL ordering — no gap, no repeat, **tied sort keys in the data**
- [ ] 6 · every mutation killed (below)
- [ ] 7 · every declared error code has at least one case
- [ ] 8 · `pnpm -F web gen:api` clean, `pnpm lint`, `pnpm typecheck`, `pnpm -F web test`
- [ ] 9 · browser pass on dev — anonymous, signed in, privileged

## Mutations

The list was committed **before** the implementation. Link that commit:

Mutation commit:

| # | Mutation | Test that went red |
|---|---|---|
| 1 |  |  |

## Census

- [ ] Callers swept, including `apps/web/server/` and `../kungal-apps`
- [ ] Production row counts recorded in the wave doc
- [ ] Visibility and permission checked per endpoint — the legacy read faces largely do not check

## After merge

Merging deploys. One at a time.

- [ ] Migration: rides the deploy / deploy-then-drop (four-step dance) / none
- [ ] Deployment verified — a 2xx from the Dokploy webhook does not mean it deployed
- [ ] Board, CHANGELOG and wave doc updated
- [ ] Worktree, branch and throwaway database removed
