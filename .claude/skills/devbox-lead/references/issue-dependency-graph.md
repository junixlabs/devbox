# Issue Dependency Graph

## Phase 0: Dogfood
```
ISS-23 (SSH Executor) ──┬──→ ISS-24 (Docker Compose) ──┐
                        │                                ├──→ ISS-26 (Workspace Manager) ──→ ISS-27 (Wire CLI)
                        └──→ ISS-25 (Tailscale)  ───────┘         ↓
                                                              ISS-28 (Unit Tests)
                                                                   ↓
                                                              ISS-29 (Integration Test)
                                                                   ↓
                                                              ISS-30 (Phase 0 Review) ★ GATE
```

## Phase 1: MVP (all blocked by ISS-30)
```
ISS-30 ──┬──→ ISS-31 (devcontainer) ─────────────────────────┐
         ├──→ ISS-32 (init) ──────────────────────────────────┤
         ├──→ ISS-33 (doctor) ────────────────────────────────┤
         └──→ ISS-34 (errors) → ISS-35 (UX) → ISS-36 (release) → ISS-37 (docs)
                                                                   ↓
         ISS-38 (Phase 1 Review) ★ GATE ←── ISS-31, ISS-32, ISS-33, ISS-37
```

## Phase 2: Multi-user (all blocked by ISS-38)
```
ISS-38 ──┬──→ ISS-39 (naming) ──────────────────┐
         ├──→ ISS-40 (port alloc) ───────────────┤
         ├──→ ISS-41 (resource limits) ──────────┤──→ ISS-44 (tests) → ISS-45 (Review) ★ GATE
         └──→ ISS-42 (server pool) → ISS-43 (multi-server) ─┘
```

## Phase 3: TUI (all blocked by ISS-45)
```
ISS-45 ──┬──→ ISS-46 (TUI) ────────────┐
         ├──→ ISS-47 (templates) ───────┤──→ ISS-50 (Review) ★ GATE
         ├──→ ISS-48 (snapshot) ────────┤
         └──→ ISS-49 (metrics) ─────────┘
```

## Phase 4: Community (blocked by ISS-50)
```
ISS-50 ──┬──→ ISS-51 (plugin) → ISS-52 (community templates) ─┐
         └──→ ISS-53 (CI/CD preview) ──────────────────────────┤──→ ISS-54 (Review) ★ GATE
```

## Phase 5: MCP & Agent Farm (blocked by ISS-54)
```
ISS-54 ──→ ISS-61 (MCP Core Tools) ──┬──→ ISS-62 (Server & Resource Tools) ──┐
                                      ├──→ ISS-63 (Snapshot & Template Tools) ┤
                                      └──→ ISS-64 (Agent Farm Sessions) ──────┤
                                                                               ↓
                                                                          ISS-65 (Tests)
                                                                               ↓
                                                                          ISS-66 (Docs)
                                                                               ↓
                                                                          ISS-67 (Review) ★ GATE
```

## Phase 6: Website & Docs (parallel with Phase 5)
```
ISS-69 (MkDocs Setup) ──→ ISS-70 (Landing Page) ──→ ISS-72 (Migration/FAQ)
         │                                                     │
         └──→ ISS-71 (User Guides) ────────────────────────────┤
                                                                ↓
                                                          ISS-73 (Review) ★ GATE
```

## Document IDs (for MCP calls)
| ISS | documentId |
|-----|-----------|
| 23 | u5fbc1o9yredqf03e2sca171 |
| 24 | bwfynbieut7neyaqqfla7nct |
| 25 | a9f6qot6x3x7fiigd2k5hbc8 |
| 26 | grsm1a99bs5gzhz75kh4fm29 |
| 27 | d04xjemipmh0zskcaj0g1f6i |
| 28 | r4eaygb3vqciy0tnuqkljhzj |
| 29 | osdowk7tui87774ccl9tgzkj |
| 30 | iob0zo0n0pd4mmtj77x33ns1 |
| 31 | hd3leohdxr7ntxnoxbin6jl6 |
| 32 | eln1qpye4zpsr5gddtolhdmb |
| 33 | hixvew1y87nazj2dmq2kz3od |
| 34 | d9j6zme3aviylrs660c4jxht |
| 35 | ednx6zytay8ttvapjxps1qg5 |
| 36 | ji7jez9uh8hqr4o26d60fso0 |
| 37 | w1u0jqovwzvqrqqqd1zlerux |
| 38 | smc9ji0y48gzpgqjaycz11oj |
| 39 | htp5ydsjiwhvwuc59jp8c8ux |
| 40 | kn5p8ubjax3txuqg16kl37yr |
| 41 | h1q8giw22c20zdtawi2h8y1v |
| 42 | cm8py6u0y5vmb8896oya6s5e |
| 43 | caiw4dkk87rdoftnja40v9am |
| 44 | n8ja241c8q1g7jw03mvp30s6 |
| 45 | gnazgkvlhyj07hmntd1llpol |
| 46 | og451yj8jo4zmf2hol04gjqi |
| 47 | tekg20g2e46lpg19g43wczka |
| 48 | o1xife9bllfwyrdcvq3i2k5d |
| 49 | u5km26hgtyyrtn08d4355qyl |
| 50 | xtdqc83mtvo5x444drodt9vg |
| 51 | sctr6nq0vb0n45v5uwac3loi |
| 52 | a8hqmlitrk2sp8q7ko88r3nr |
| 53 | p45ldoiyrmn3tc3ucqdnfx7t |
| 54 | uz93t6ov36id0qxiylt8442r |
| 61 | vodbmcqjoivt4urctmfntkgm |
| 62 | bj86l32s9ufog3pllpes9006 |
| 63 | khw47k3zn4gxwre3sg0q0o4w |
| 64 | f5pvcc11f6gsq6cboxa0c4gl |
| 65 | apdrgmt5c2afrc9x7z79y6ds |
| 66 | sbfna8dv8n66nuf2kd9csssw |
| 67 | yo6e3q3vq0h6pafvgl2rjfcx |
| 69 | orzxxbzjryowdynxtp667z94 |
| 70 | jd8n8phlcd8ijrs3kv01mo2k |
| 71 | dh2w92is9f7agdat9k4486m7 |
| 72 | fn90qhfoe10scamk9v56rm6i |
| 73 | r5ins098mlsfgxx2lz3mqfy8 |
