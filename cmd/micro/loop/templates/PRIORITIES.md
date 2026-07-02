# Priorities

A single ranked queue, highest-value first. Each item links a scoped issue the
loop can build and CI can verify. The **planner** keeps this current; the
**builder** takes the top item whose issue is still open.

<!--
Seed this with a few real items to give the loop a running start, e.g.:

1. Add retry with backoff to the HTTP client — #123
2. Document the config file format — #124
3. Fix flaky timeout in the cache tests — #125

The planner will re-rank, drop completed items, and file issues for new gaps.
Reorder or edit this file at any time to redirect the loop.
-->
