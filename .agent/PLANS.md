# Long-Term Plan (Performance Investigation)

- Note missing long-term memory file ./.agents/PLAN.md and prepare to recreate after planning.
- Survey repository structure and identify core performance-critical components (reflector, schema generation).
- Locate existing benchmarks or performance-related tests; catalog gaps if none found.
- If benchmarks are absent, design representative benchmarks that exercise schema generation hot paths without altering APIs.
- Review previous PLAN.md expectations to ensure alignment once recreated.
- Run go test -bench=. -benchmem ./... to capture baseline metrics (even if zero benchmarks).
- Persist baseline benchmark output to a file for later benchstat comparison.
- Profile code paths conceptually using code inspection to hypothesize bottlenecks (reflection caching, type processing, map allocations).
- Use gopls to summarize packages or highlight heavy functions for targeted review.
- Open key source files (reflect.go, schema.go, utils.go) to understand existing caching and reflection logic.
- Identify allocation-heavy spots (e.g., repeated tag parsing) suitable for low-risk optimization.
- Draft candidate optimizations that preserve API compatibility (e.g., reuse parsed tags).
- Select one or two high-impact, low-risk optimizations to implement first.
- Implement chosen optimization in code with minimal diff and clear comments if needed.
- Add or refine benchmarks to cover the optimized path ensuring representative workloads.
- Run go test -bench=. -benchmem ./... to collect post-change metrics.
- Run benchstat comparing baseline vs post-change benchmark outputs to quantify impact.
- Evaluate results; if degradation occurs, reconsider or revert changes.
- Ensure API compatibility and no behavior changes in public surface; run full go test ./... if needed.
- Document findings and updated long-term memory to ./.agents/PLAN.md.
- Summarize performance improvements, benchmark data, and next steps for user.
- If optimizations succeed, propose further potential areas for future work.
- Clean up temporary files (benchmark outputs) if not needed.
- Confirm compliance with coding guidelines and lint expectations.
- Deliver final Japanese-language report with actionable insights and next actions.

## Status (2025-12-02)
- Added `reflect_bench_test.go` with `BenchmarkReflectTestUser` to cover tag-heavy struct reflection.
- Optimized tag handling by parsing `json`, `jsonschema`, and `jsonschema_extras` once per field and reusing the slices.
- Baseline vs optimized (darwin/arm64, Apple M3 Max, `-count=5`):
  - `ReflectTestUser`: 48.65µs → 45.55µs (-6.38%, p=0.008), 54.57KiB → 53.87KiB (-1.28%, p=0.008), 654 → 638 allocs/op (-2.45%, p=0.008).
- Full test suite passes: `go test ./...`.
- Bench artifacts cleaned from workspace; benchstat installed via `go install golang.org/x/perf/cmd/benchstat@latest`.
- Second pass (2025-12-02): Added more benchmarks and global field-tag cache using xsync.MapOf.
  - Baseline vs optimized (darwin/arm64, Apple M3 Max, `-count=10`):
    - `ReflectTestUser`: 46.73µs → 36.07µs (-22.81%), 53.87KiB → 49.04KiB (-8.96%), 638 → 444 allocs/op (-30.41%).
    - `ReflectEmbedded`: 3.680µs → 3.197µs (-13.13%), 5.75KiB → 5.48KiB (-4.63%), 57 → 41 allocs/op (-28.07%).
    - `ReflectDoNotReference`: 3.457µs → 2.980µs (-13.81%), 5.04KiB → 4.77KiB (-5.32%), 55 → 39 allocs/op (-29.09%).
    - `ReflectExpandedStruct`: 3.763µs → 3.313µs (-11.96%), 5.75KiB → 5.49KiB (-4.59%), 57 → 41 allocs/op (-28.07%).
    - `ReflectWithComments`: 4.002µs → 3.573µs (-10.72%), 6.08KiB → 5.82KiB (-4.38%), 64 → 48 allocs/op (-25.00%).
  - geomean improvement: -14.60% sec/op, -5.59% B/op, -28.15% allocs/op.
  - Changes: added benchmarks for embedded/comment/DoNotReference/ExpandedStruct cases; introduced global field tag cache keyed by (type, field tag name) using `xsync.MapOf`; kept API surface unchanged.
  - Pending: optional schema cache, further preallocation tweaks, micro tag parsing.

## Next Focus
- Add more benchmarks covering embedded/alias-heavy structs and comment extraction paths to tighten confidence intervals.
- Explore further allocation cuts (e.g., reusing ordered map instances or preallocating Required slices) without API changes.
- Consider caching parsed tags per struct type if we can prove thread safety and no cross-call mutation risks.

## Status (2025-12-02 later pass)
- Optimized schema tag parsing by switching to `strings.Cut`/manual slices to avoid temporary slice allocations; removed default allocations for unprocessed tag buffers; preallocated `Required` slice per struct.
- Baseline (pre-change, single sample):
  - ReflectTestUser 35.30µs 48.97KiB 444 allocs/op.
  - ReflectEmbedded 3.148µs 5.480KiB 41 allocs/op.
  - ReflectDoNotReference 2.938µs 4.769KiB 39 allocs/op.
  - ReflectExpandedStruct 3.243µs 5.485KiB 41 allocs/op.
  - ReflectWithComments 3.513µs 5.812KiB 48 allocs/op.
- Post-change (single sample):
  - ReflectTestUser 30.05µs 44.84KiB 285 allocs/op.
  - ReflectEmbedded 2.948µs 5.332KiB 31 allocs/op.
  - ReflectDoNotReference 2.746µs 4.625KiB 29 allocs/op.
  - ReflectExpandedStruct 3.025µs 5.335KiB 31 allocs/op.
  - ReflectWithComments 3.433µs 5.673KiB 38 allocs/op.
- Benchstat (n=1 each, needs more samples) shows ~7.5% geomean time improvement and ~26% alloc reduction; rerun with `-count=5` for stable stats if needed.
- All tests pass (`go test ./...`).

## Updated Next Focus
- Rerun benchmarks with `-count=5` before/after to firm up confidence intervals and update PLAN metrics.
- Consider further trimming allocations in ordered-map usage or optional schema caching gated behind a flag to preserve API compatibility.

## Status (2025-12-03)
- Scope: core reflection-based schema generation; Focus: CPU/alloc throughput while keeping API stable.
- Opportunities identified: (1) preallocate `properties` ordered-map based on struct field count to avoid growth reallocations, (2) rewrite `splitOnUnescapedCommas` as a single-pass parser to cut allocations/copies in tag handling, (3) optional per-Reflector schema memoization keyed by config snapshot to skip repeated reflection for identical types, (4) revisit `appendUniqueString` dedupe cost only if large-struct benchmark shows it matters.
- Pending measurements: fresh baseline `go test -bench=Reflect -benchmem -count=5 ./...` with artifacts for benchstat; add wide-struct and repeated-reflection benchmarks to stress new optimizations.
- Next actions: design capacity plumbing for `NewProperties`, implement tag-split parser and accompanying tests, prototype safe per-Reflector schema cache, rerun benches and benchstat, document impact and clean artifacts.

## Status (2025-12-03 later)
- Scope (inferred): core reflector/schema generation pipeline; Focus (inferred): CPU time and allocations/throughput without API changes.
- Fresh findings: `orderedmap.New` accepts a capacity hint—can preallocate `Properties` to `NumField` to avoid rehash/reslice; `splitOnUnescapedCommas` currently double-splits; schema cache would need config-key and thread-safe map (xsync) plus invalidation story.
- Selected strategies to pursue next: (1) plumb capacity into `NewProperties` and call with struct field count; (2) rewrite `splitOnUnescapedCommas` as a single-pass parser and extend tests; (3) prototype optional per-Reflector schema memoization keyed by reflector config + type, guarded to keep API behavior stable.
- Next concrete actions: capture fresh baseline `go test -bench=Reflect -benchmem -count=5 ./...` artifacts; add wide-struct and repeated-reflection benchmarks; implement capacity and tag-split improvements; prototype schema cache behind a feature flag; run benchstat pre/post; update docs/notes and clean artifacts.

## Status (2025-12-03 evening)
- Scope/Focus (inferred): reflector/schema generation; CPU + alloc throughput, API-compatible.
- Implemented: `NewProperties` now accepts optional capacity hint and `reflectStruct` passes `NumField()`; `splitOnUnescapedCommas` rewritten single-pass to remove temp slices; added edge-case tests; added benchmarks for wide struct (32 fields) and repeated type reflection.
- Baseline cmd: `go test -bench=Reflect -benchmem -count=5 ./... > /tmp/bench_before.txt`; Post-change: same into `/tmp/bench_after.txt`; benchstat run.
- Benchstat highlights (n=5, Apple M3 Max): `ReflectTestUser` -7.66% sec/op, -3.42% B/op, -1.75% allocs/op (p=0.008); others ~flat; new wide-struct benchmark: ~17.7µs, 29.5KiB, 122 allocs/op establishing target for further cuts.
- Tests: `go test ./...` passes.
- Remaining opportunities: per-Reflector schema memoization (design key + invalidation, default off), possible further preallocation/ordered-map reuse if wide-struct numbers remain high, revisit appendUniqueString only if large struct benchmarks show it matters.
- Next actions: design + prototype schema cache flag; consider preallocating Required/OneOf arrays for wide structs; rerun benches/benchstat; clean bench artifact files when done.

## Status (2025-12-04 early)
- Added optional per-Reflector schema cache (`EnableSchemaCache`) with fingerprinting of config and cloning to avoid mutation sharing; cached via xsync.MapOf. Fingerprint includes base ID, tag config, bool flags, function pointers, and CommentMap pointer/len.
- Added deep clone helpers for Schema (maps, slices, ordered properties) to safely serve cached copies.
- Tests added: cache isolation (mutating one result doesn’t leak), config fingerprint variation (BaseSchemaID change yields different IDs). All tests pass.
- Pending: evaluate cache performance/footprint with `ReflectRepeatedType` benchmark (cache on/off), decide default flag exposure in docs, clean /tmp benches, and update long-term stats if measurable gains appear for repeated reflection workloads.

## Status (2025-12-04 later)
- Implemented optional cache size bound + LRU promotion; cache still default-off. Added `BenchmarkReflectRepeatedTypeCachedLimited` to measure capped-cache scenarios.
- Tested higher-count benches (count=10/20, benchtime=2x/5x). Findings: cache ON or capped can improve sec/op on repeated reflections but increases B/op; WideStruct unaffected. Due to noise and mixed trade-offs, kept cache default-off and left further tuning for opt-in users.
- Tried unique/weak approaches; reverted due to regressions (unique) or correctness concerns (shallow cache). Current code mirrors commit 40eac9a with LRU additions.
- Next actions: (1) document cache usage guidance (when to enable, set MaxSchemaCacheEntries), (2) consider conditional Required prealloc threshold tuning if small structs show regressions, (3) clean up bench artifacts when not needed.
