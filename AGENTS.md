Use the `cocoindex-code-rs` MCP server automatically for semantic code search when:
- the user asks by behavior, intent, or meaning rather than exact text
- the codebase area is unfamiliar
- similar implementations or related patterns are needed
- grep, filename search, or symbol lookup is noisy or inconclusive

Prefer normal text search first when exact names, symbols, routes, config keys, or error strings are known.

When using `cocoindex-code-rs`:
- use it to identify candidate files and code chunks
- then verify results by reading files or using local text search
- avoid repeated semantic searches if one search already narrowed the area
- when working in a monorepo, prefer limiting search to the current subproject or language before searching the whole repository
