# Native article extraction evaluation

Date: 2026-07-20

Decision: use `codeberg.org/readeck/go-readability/v2` v2.1.2 as the primary
HTML extractor and retain the bounded regex extractor only as a fallback.

The selected library is the maintained successor recommended by the deprecated
go-shiori package, follows Mozilla Readability.js 0.6 behavior, has a tagged
v2 release, and is MIT licensed. Learnloom passes already-downloaded bytes to
`FromReader`; the library never performs its own network request.

## Fixture comparison

`TestReadabilityFixtureEvaluation` compares eight deterministic saved-layout
fixtures. Character counts are normalized extracted text:

| Fixture | Readability | Previous fallback | Observation |
|---|---:|---:|---|
| Semantic article | 1355 | 1355 | Equivalent main content |
| Main documentation | 1363 | 1363 | Equivalent main content |
| Content-class layout | 1356 | 1370 | Sidebar noise removed |
| Blog post | 1354 | 1379 | Menu/comment noise removed |
| Nested sections | 1365 | 1365 | Equivalent main content |
| Metadata and scripts | 1364 | 1364 | Script content excluded |
| Utility-heavy page | 1358 | 1372 | Toolbar noise removed |
| Reference with table | 1386 | 1386 | Reference table retained |

All fixtures preserved their marked explanatory content and excluded active
script text. The focused improvement on non-semantic content layouts justifies
the dependency; fallback remains useful for unusually short or malformed
pages that Readability declines.
