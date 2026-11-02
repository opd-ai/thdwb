# AUDIT_REPORT.md — thdwb (The Hotdog Web Browser): Comprehensive Technical Audit

**Audit date:** 2026-11-01 · **Auditor:** independent re-audit (this document supersedes prior in-repo audit drafts)
**Repository:** `github.com/opd-ai/thdwb` (module `github.com/danfragoso/thdwb`)
**HEAD audited:** `33eebcd` ("fix(extensions): close implementation gaps between PLAN.md and codebase"), 562 commits
**Environment:** Linux amd64, Go 1.23.10 host → go1.26.0 toolchain (GOTOOLCHAIN=auto), X display :0, staticcheck 2025.1.1 (go1.26 build)

**Method:** All promises extracted from the *current* README.md and PLAN.md; every material claim verified by **execution** (build, full test suite, race detector, static analysis, live GUI runs, headless engine probes against real hosts, pixel-level render measurement, binary linkage inspection) plus file:line code citation. Full evidence: FUNCTIONAL_TEST_RESULTS.md.

---

## 1. Executive Summary

**Verdict: PARTIALLY.** thdwb does **not** deliver a "working, usable" browser, but it **does** deliver what its README actually promises for 2026 — a pure-Go, well-tested, well-documented *learning tool* with substantial (if disconnected) browser-engine components and a broad in-process WebExtensions platform.

| Question | Answer | Confidence |
|---|---|---|
| Working, usable browser? | **NO** — binary panics on first render frame (2/2 repro); 100% of real-world fetches fail; no code path displays web content | HIGH (execution-verified) |
| Pure Go? | **YES** — no CGo in source; binary links only libc/libdl/libpthread; Ebitengine via purego; GL dlopen'd at runtime | HIGH |
| Great learning tool? | **YES** — 7 named subsystems as tested libraries (63–97% cov), 1,023 test functions, real textbook, complete docs | HIGH |
| "100% Firefox WebExtensions compatible" (PLAN.md)? | **NO** — 18 chrome.* namespaces implemented & unit-tested; 66% of E0–E12 checklist items genuinely delivered, 29% partial/stub/isolated, 5% unverifiable; absolute success criteria (top-100 extensions, 99.9% uptime, no crashes) unmet | HIGH |

**The one-sentence technical finding:** the browser is a collection of individually-tested, individually-reasonable components — network, parser, CSS, layout, graphics, JS, UI, extensions — **whose interconnections are either missing or broken**: the runtime parse path bypasses the "own" HTML parser, CSS and layout are never invoked, the layout engine renders zero pixels even when forced, the network transport fails on every non-local host and swallows the error, the UI crashes on frame 1, and the content area is a literal placeholder.

**Maturity rating: proof-of-concept** for the browser product; **early-alpha quality** for the component libraries and extension platform; **beta quality** for documentation and tooling. (Details: GAP_ANALYSIS.md §4.)

**Why the test suite is green while the product is broken:** tests exercise *library internals* (fixtures, localhost httptest servers, in-process state managers, geometry invariants) — none of the 1,023 test functions launches the binary, fetches a real host, or asserts a rendered pixel. The four critical defects each sit exactly on the seams between components, which no single-component test covers.

---

## 2. Project Overview

**What it is (today):** a single-Go-module repository (~28k non-test LOC, 253 Go files) implementing, per its three-layer design (PLAN.md:599-612):
1. **thdwb library** — engine components: `ketchup` (HTML→DOM adapter, 469 LOC), `mayo` (CSS parse/style, 1,023), `bun` (layout/paint, 795), `gg` (2D graphics, 1,838), `sauce` (network/cookies/HSTS/CORS/TOFU, 2,232), `hotdog` (DOM + goja JS runtime + storage + CSP, 5,495), `mustard` (UI toolkit on Ebitengine, 3,249), plus `webview`/`platform` (dead code) and `profiler`.
2. **Extension runtime** — `browser/extensions/` (53 files): WebExtensions API facade (18 `chrome.*` namespaces), manifest v2/v3 parser, container manager, sandboxes, xpi/updater, service workers, crash recovery, multiprocess (unwired), CDP-style debug server (unwired), profiler, test harness.
3. **Browser shell** — `browser/shell/` (40 files): tab manager, address bar (autocomplete + security indicator), menu bar, bookmarks/history sidebars, settings, new-tab page with container selector, extension toolbar, session restore, keyboard shortcuts.

**Stated goals (current README.md):** "from-scratch web browser written entirely in Go" with own ketchup/mayo/bun/gg/mustard/sauce/hotdog; dependencies "primarily Ebitengine, extant third party Go libraries preferred"; **"Not stable, not spec-compliant, not production-ready — but a great learning tool!"** (README.md:16).
**Stated goals (current PLAN.md):** "freestanding consumer browser … 100% Firefox WebExtensions API compatibility, including the Container Tabs API", phases E0–E12, all checklists `[x]`.

**Evolution note:** the repo has grown far beyond the audit brief's description (which referenced an older 8-phase engine PLAN and an "only OpenGL and GLFW" README). The audit validates the *current* tree; where the brief and the tree disagree, the tree wins (and the discrepancies are recorded in BASELINE.md).

---

## 3. Feature Matrix (condensed — full version: FEATURE_MATRIX.md)

### Engine (README R1–R8)
| Component | Status | Key evidence | Maturity |
|---|---|---|---|
| ketchup (own HTML parser) | **Disconnected** — live path uses `x/net/html` (hotdog/node.go:783); 0 non-test importers | 97.3% cov as lib | lib beta / wiring absent |
| mayo (own CSS engine) | **Disconnected** — reachable only via ketchup; live DOM unstyled; `querySelector` always null (node.go:414) | 95.9% cov as lib | lib beta / wiring absent |
| bun (own layout) | **Disconnected + non-functional** — 0 importers; **0 visible pixels** when forced (zero-width boxes non-exp; zero-height exp) | 88.0% cov as lib | broken-as-renderer |
| gg (2D graphics) | Works for UI chrome | 63.5% cov | experimental |
| mustard (UI toolkit) | **Crashes frame 1** (frame.go:90); orientation branches transposed (widget.go:46-82) | 77.3% cov | broken at runtime |
| sauce (network) | **Broken for all non-local hosts** (double transport wrap); errors swallowed as success; localhost fine | 78.7% cov | beta (localhost) / broken (web) |
| hotdog (JS runtime) | Works headlessly; `document.title` = function (js_bindings.go:184); querySelector null; unstyled DOM | 82.8% cov | beta (binding defects) |
| webview / platform | **Dead code** (0 importers; GPUInfo hardcoded) | — | placeholder |

### WebExtensions (PLAN E0–E12 — all checklists `[x]`)
| Phase | Delivered | Notes |
|---|---|---|
| E0–E3 (arch, messaging, containers, scripting/menus/permissions/alarms/i18n) | 75–82% | in-process, unit-tested, passing; containers proven at map level only |
| E4 (WebRequest/notifications/downloads) | 57% | downloads write real files (via `http.DefaultClient` — the one working network path); notifications in-memory only |
| E5 (content-script sandbox) | 50% | per-extension shared VM (not per-page); isolation test simulates page with a second sandbox |
| E6 (xpi/install/update) | 80% | signature "validation" = META-INF detection only, no crypto verify |
| E7 (browser UI) | 75% as UI code / **0% usable** | full chrome builds + unit-tests; crashes at runtime; no page display |
| E8 (perf/testing/stability) | 64% | components real + wired; "major websites render" claim supported only by local fixtures |
| E9 (release) | 50% | docs excellent; gallery/installer/autoupdate isolated; external infra unverifiable |
| E10 (Manifest v3/sync/WebDriver) | 44% | SW lifecycle real; **storage.sync = Noop stub**; WebDriver unwired |
| E11 (scale) | 25% | tab suspension wired; **multiprocess unwired** (13.8% cov, 0 consumers) |
| E12 (DX) | 50% | hot reload + profiler real; debug server unwired |

**Totals: 67/102 items (66%) fully delivered · 30 (29%) partial/stub/isolated · 5 (5%) unverifiable. "100% compatible": not met.**

---

## 4. Critical Gaps and Limitations (prioritized)

**P0 — blocks the product's core purpose (all execution-verified):**
1. **Startup panic** — `mustard/frame.go:84-92`: child layout computed from visible-only set, indexed by the full child list → `panic: index out of range [6] with length 6` on frame 1, every launch, settings-independent (reproduced 2/2). Fix: iterate the *same* list the layout was computed from (or skip hidden children consistently).
2. **100% real-world fetch failure** — `sauce/sauce.go:949` + `:968` call `requestClient` twice, so the inner "base" transport is an `*originCookieTransport`; `newVerifiedTransport`'s `*http.Transport` assertion (sauce/certificate_validation.go:36-39) fails for every HTTPS request — and `upgradeHTTPURL` forces every non-localhost `http://` to `https://`. Measured failure: `Get "https://example.com": configure HTTPS transport: HTTPS requires a configurable *http.Transport, got *sauce.originCookieTransport` in <300 µs. Fix: wrap once (drop the second `requestClient` before `.Do`), or make `newVerifiedTransport` accept an already-wrapped base.
3. **Silent failure swallowing** — on fetch error, `Resource.Code` stays 0 (sauce/sauce.go:965-970); `LoadURL` rejects only `>= 400` (window_context.go:342). DNS/TLS/timeout → "page loaded successfully" with an internal error HTML. Fix: set a sentinel Code (e.g., 0 → propagate error) and surface it to the shell.
4. **No web content rendering** — shell content area is a placeholder (shell.go:344-347); `bun.RenderDocument` has 0 call sites; `WindowContext` has no render method; `webview` has 0 importers. Fix: create a per-tab `WindowContext`, call `LoadURL`, then `bun.RenderDocument` into the tab's surface (the `webview` package was evidently built for this and can be revived).

**P1 — core subsystem defects (fix required before P0 items are meaningful):**
5. Live parse path bypasses ketchup; `HTMLRoot`/`StyleSheets` never set → **CSS never applied; `querySelector`/`querySelectorAll` always null in page JS** (node.go:414,448; ketchup/parser.go:20; mayo/mayo.go:397-409). Fix: route `LoadHTML` through ketchup (or set HTMLRoot/StyleSheets in the x/net path).
6. `bun` renders **0 pixels** on both paths: zero-width boxes (width inheritance from un-laid-out parent, bun/bun.go:11-20), zero-height flex boxes, and `paintBlockElement` sets color after `DrawRectangle` (bun/block.go:13-16). Add pixel-output tests (they would have caught all three).
7. Mustard orientation branches transposed (widget.go:46-82) — chrome layout wrong even post-panic-fix.
8. `document.title` bound as a function (js_bindings.go:184) — use `DefineAccessorProperty` as done for `body`.
9. Content-script sandbox: one shared VM per extension across tabs (manager.go `SandboxFor`) vs PLAN E5's per-page isolation; isolation test simulates the page (content_script_sandbox_test.go:183-198).
10. `.xpi` signature "validation" does no cryptographic verification (xpi.go:105-111).

**P2 — checklist-vs-reality gaps (stubs/isolated):** storage.sync Noop (storage_sync.go:27-43) · multiprocess unwired · WebDriver/gallery/installer/debug-server unwired · notifications in-memory · webview/platform dead.

**P3 — hygiene:** 25 staticcheck findings (mostly U1000 dead code) · stale README prereqs (GLFW/go-gl, Go 1.21+) · stale Makefile "workspace" comment · go.mod replace block vestigial · CI go-version 1.25 vs go.mod 1.26.0 · `OPTIMIZED_AUDIT_PROMPT.md` + stale `coverage_*.out` committed in tree · textbook doesn't disclose the disconnected pipeline.

**Explicit non-gaps (by design, per README.md:16 / PLAN.md:596-597):** not spec-compliant, not production-ready, mobile out of scope, browser-level (not OS-level) container isolation, unsigned MVP gallery.

---

## 5. Code Quality Assessment (full detail: QUALITY.md)

- **Error handling:** sensible typed-error model at library level; four defects break it in the product (crash, swallowed fetch errors, silent layout degeneracy, discarded assignment at extension_ops.go:161).
- **Static analysis:** `go vet` clean; staticcheck 25 findings (mostly dead code — corroborating the unwired-scaffolding finding); no data races under `-race` (29 packages).
- **Tests:** 1,023 functions, all green; combined coverage 69.1% (per-package 13.8%–100%). Coverage measures internals, not behavior: no test launches the binary, fetches a real host, or asserts pixels. The "website rendering integration tests" (E8) run local fixtures only.
- **Logging/debugging:** profiler, console collector, inspector, network monitor, error logger, opt-in telemetry — all real, tested, and wired into the shell; currently low practical value (nothing to observe until P0 is fixed).
- **Modularity:** clean package boundaries, good interface seams (Driver, TabOperations, SyncProvider…), no import cycles; the *wiring* between layers is where it fails.
- **Documentation:** genuinely excellent — 8-chapter textbook at claimed reading level, migration guide, release/security/support docs, per-package API comments. Accuracy gap: describes the intended pipeline, not the disconnected reality.

---

## 6. Functional Test Results (condensed — full outputs: FUNCTIONAL_TEST_RESULTS.md)

| Check | Result |
|---|---|
| `make build` (CGO_ENABLED=0) | ✅ 21.6 MB binary, ~60 s; `ldd`: libc/libdl/libpthread only |
| `make test` | ✅ 28 packages ok, 0 fail |
| `go test -race work` | ✅ 29 packages, no data races |
| `go vet ./...` | ✅ clean |
| staticcheck 2025.1.1 (go1.26) | ⚠️ 25 findings |
| Coverage (fresh) | 69.1% combined |
| **Live run `bin/thdwb` (×2, ±settings)** | ❌ **panic frame.go:90, first frame, no window** |
| ketchup→mayo→bun render probe | ❌ **0 non-white pixels** (both layout paths) |
| hotdog.LoadHTML probe | ⚠️ DOM+JS work; title=function, querySelector=null, unstyled |
| Real-host fetch probes (example.com, 404, .invalid, self-signed.badssl.com) | ❌ all fail <300 µs (transport bug); **LoadURL returns nil error** for unreachable/404 |
| Malformed HTML | ✅ no panic, DOM built |
| SOP/CSP/container-cookie unit suites | ✅ pass (live CORS blocked by transport bug) |
| Tabbed/multi-window live | ⛔ unverifiable (crash); state-level tests pass |

---

## 7. Maturity Rating

**Overall: proof-of-concept (browser) · early-alpha (component libraries + extension platform) · beta (docs/tooling).**

Justification: an alpha browser must launch and display a page — thdwb's binary does neither (P0 findings 1–4, execution-verified). But it is far beyond a toy: ~28k LOC of real, tested subsystems across all seven named engine components, a broad unit-tested WebExtensions platform, complete documentation, CI with race tests, and zero data races. The gap between "what the libraries can do" and "what the product does" is entirely in the composition layer — which is precisely where the fixes needed (P0/P1) are small, well-localized, and each individually verifiable.

---

## 8. Recommendations

**For maintainers (in dependency order — each step unlocks the next):**
1. **Fix the frame-draw panic** (mustard/frame.go:84-92: iterate the visible-only list the layout was computed from) + **swap the transposed orientation branches** (widget.go:46-82). Add a mustard test that renders a frame with hidden children (would have caught #1).
2. **Fix the sauce transport** (remove the second `requestClient` wrap at sauce/sauce.go:968, or make `newVerifiedTransport` accept wrapped bases) and **stop swallowing fetch errors** (propagate a non-2xx sentinel Code / return an error from `LoadURL`). Add one sauce test against a non-localhost https host (via `httptest.NewUnstartedServer` + TLS, or a real-host smoke test marked integration).
3. **Connect the pipeline:** per-tab `WindowContext` in the shell (`TabManager.Navigate` → `LoadURL`), then `bun.RenderDocument` into the tab surface — the `webview` package (0 importers) appears designed for exactly this; revive it. Route `LoadHTML` through ketchup (or set `HTMLRoot`/`StyleSheets`) so CSS and `querySelector` work.
4. **Make bun actually paint:** set colors before paths (bun/block.go:13-16), fix width/height inheritance (bun/bun.go:11-33), and **add pixel-output tests** (render → count non-white pixels / sample colors). The bun suite is 88% covered and still renders nothing — pixel assertions are the missing invariant.
5. **Fix the `document.title` binding** (js_bindings.go:184 → `DefineAccessorProperty`, mirroring `body`).
6. **Reconcile PLAN.md checklists with reality:** mark E10 storage.sync (Noop), E11 multiprocess (unwired), E10/E12/E9 unwired servers as in-progress, or wire them in; replace "signature validation" with real verification (ed25519 over the checksum) or rename it "signature detection".
7. **Hygiene pass:** remove dead code flagged by staticcheck (25 findings), delete `OPTIMIZED_AUDIT_PROMPT.md` and stale `coverage_*.out` from the tree, fix README prereqs (drop GLFW/go-gl; Go 1.26+), Makefile workspace comment, go.mod replace block, and align CI go-version with go.mod.
8. **Add one true end-to-end test** (opt-in, network): launch the engine headless → `LoadURL` a fixture server on 127.0.0.1:port (note: localhost is *not* upgraded, so it already works) → assert DOM + rendered pixels. This single test would have caught findings 1–6.

**For potential users:**
- **Do not use as a browser** — it cannot launch, fetch, or display.
- **Use as a study/reference resource** — the seven named subsystems, the textbook (docs/01–08 + glossary), and the per-module tests are a coherent, high-quality map of browser internals; run `make test` and read the chapters alongside the code.
- If you embed the engine: use the **library APIs** (`hotdog.WindowContext` + `ketchup`/`mayo`/`bun` wired as shown in browser/integration_test.go) rather than the `bin/thdwb` app, and expect the P1 defects until fixed.

**For contributors:**
- The highest-leverage work is P0/P1 (all well-localized, each with an obvious regression test). The extension platform is the most mature surface if you want to contribute there (50.2% coverage in `browser/extensions` is the biggest coverage gap among active code).
- Match the project's conventions: per-package doc comments, testify, small focused tests, English commit messages with `type(scope):` prefixes.

---

## 9. Appendices

The full evidence base for this report (all in repository root):

| Appendix | Contents |
|---|---|
| **BASELINE.md** | Master promise checklist (README R1–R17, PLAN P0–P15) with expected locations, validation methods, dependency catalog, pure-Go verification, structural facts |
| **ARCHITECTURE.md** | Module topology, dependency tree, 3-layer design, advertised-vs-actual request-to-render flow with import-graph evidence, entry points, error-handling & logging assessment |
| **FEATURE_MATRIX.md** | Per-component and per-phase (E0–E12) status → evidence → maturity, patterns, README cross-reference |
| **QUALITY.md** | Error-handling review, build/test/static-analysis results, staticcheck findings, coverage tables, documentation assessment, 12-item technical-debt inventory |
| **FUNCTIONAL_TEST_RESULTS.md** | Full captured outputs: build log, test/race/vet/staticcheck results, live-run panic transcripts (×2), 4 headless probes, edge cases, security-boundary matrix, stability observations |
| **GAP_ANALYSIS.md** | Quantified promise-vs-working tables, finding categorization (critical/major/moderate/minor), by-design vs incomplete, maturity rating, verdict with confidence |

**Reproduction artifacts (this machine):** `/tmp/thdwb_build.log`, `/tmp/thdwb_test.log`, `/tmp/thdwb_race.log`, `/tmp/thdwb_run.log`, `/tmp/thdwb_run2.log`, `/tmp/engineprobe_output.txt`, `/tmp/thdwb_staticcheck.log`, probe sources `/tmp/engineprobe{,2,3}/`, rendered PNGs `/tmp/engineprobe3/*.png`.

**Verdict restated: PARTIALLY — not a working browser (execution-verified, high confidence); a genuine pure-Go learning tool with strong, disconnected, well-documented components.**
