# Trim Difference Design

Date: 2026-07-13
Status: Approved design, awaiting implementation planning

## Context

The library needs to identify the single continuous fragment changed between an
original string and a revised string. The operation removes the exact common
rune prefix and suffix, returns the remaining fragments, and reports the
original fragment's rune offsets.

Insertion-only changes leave the minimal original fragment empty. By default,
the result must include one unchanged context rune so consumers can still work
with a non-empty original range. Callers may explicitly request the minimal
empty range instead.

## Goals

- Compare two strings by exact rune equality.
- Remove their longest non-overlapping common rune prefix and suffix.
- Return the trimmed original and revised fragments.
- Return a half-open rune range `[Start, End)` in the original input.
- Default to padding an empty original fragment with one context rune.
- Allow callers to preserve an empty original fragment instead.
- Return stable, distinguishable errors for invalid input and no change.
- Keep the public API consistent with the existing root package.

## Non-goals

- Computing multiple disjoint edits.
- Producing an edit script or semantic diff.
- Normalizing case, whitespace, punctuation, or Unicode forms.
- Adding CLI support in this change.
- Reorganizing existing unrelated capabilities.

## Public API

The API will be added to the root `textprocessor` package in `trim.go`.

```go
type TrimResult struct {
	Original string `json:"original"`
	Revised  string `json:"revised"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
}

type TrimEmptyOriginalStrategy string

const (
	TrimEmptyOriginalPad  TrimEmptyOriginalStrategy = "pad"
	TrimEmptyOriginalKeep TrimEmptyOriginalStrategy = "keep"
)

type TrimOptions struct {
	EmptyOriginal TrimEmptyOriginalStrategy `json:"empty_original"`
}

var (
	ErrOriginalEmpty      = errors.New("original must not be empty")
	ErrNoChange           = errors.New("original and revised have no change")
	ErrInvalidTrimOptions = errors.New("invalid trim options")
)

func TrimDifference(
	original, revised string,
	opts ...TrimOptions,
) (TrimResult, error)
```

The zero option value defaults `EmptyOriginal` to
`TrimEmptyOriginalPad`. `TrimEmptyOriginalKeep` must be specified explicitly.
As with the package's existing optional-options APIs, the first options value
is used when supplied.

Unknown non-empty strategy values return an error that matches
`ErrInvalidTrimOptions` through `errors.Is`.

## Result Contract

For every successful result:

```text
0 <= Start <= End <= utf8.RuneCountInString(original)
Original == string([]rune(original)[Start:End])
```

Replacing the original rune range `[Start, End)` with `Revised` reconstructs
the complete `revised` input. `Revised` does not have a separate offset range.

Offsets include the context rune when padding is applied. All offsets are rune
offsets, not byte offsets, matching the existing `Span`, `Block`, and
`MatchResult` public types.

## Validation and Errors

Validation uses this precedence:

1. If `original` contains no runes, return `ErrOriginalEmpty`.
2. If `original` and `revised` have identical rune sequences, return
   `ErrNoChange`.
3. If the selected empty-original strategy is invalid, return an error matching
   `ErrInvalidTrimOptions`.

Consequently, two empty strings return `ErrOriginalEmpty`. An empty `revised`
is valid when `original` is non-empty and represents deletion.

## Algorithm

1. Convert `original` and `revised` to rune slices.
2. Validate the non-empty original and reject unchanged input, then resolve the
   selected strategy.
3. Scan from the beginning to find the longest exact common rune prefix.
4. Scan backward to find the longest exact common rune suffix. Stop before the
   suffix overlaps the already removed prefix in either input.
5. Set the minimal original range to the runes between the common prefix and
   suffix. Set the revised fragment to its corresponding runes between the same
   common regions.
6. If the minimal original fragment is non-empty, return it unchanged.
7. If it is empty and the strategy is `TrimEmptyOriginalKeep`, return the empty
   fragment and its zero-length insertion range.
8. If it is empty and the strategy is `TrimEmptyOriginalPad`:
   - Prefer the last rune of the common prefix. Expand both fragments to the
     left by one rune and expand the original range accordingly.
   - If there is no left context, use the first rune of the common suffix.
     Expand both fragments to the right by one rune and expand the original
     range accordingly.

`original` is guaranteed non-empty and unchanged inputs are rejected, so an
insertion-only result always has either left or right original context available
for the default padding strategy.

The algorithm is `O(n+m)` in time and uses `O(n+m)` space for rune slices.

## Examples

Default `TrimEmptyOriginalPad` behavior:

| Original input | Revised input | Result Original | Result Revised | Range |
| --- | --- | --- | --- | --- |
| `abc` | `abXc` | `b` | `bX` | `[1,2)` |
| `abc` | `Xabc` | `a` | `Xa` | `[0,1)` |
| `abc` | `abcX` | `c` | `cX` | `[2,3)` |
| `abc` | `axc` | `b` | `x` | `[1,2)` |
| `abc` | `ac` | `b` | empty | `[1,2)` |

For `original="abc"`, `revised="abXc"`, and
`TrimEmptyOriginalKeep`, the result is `Original=""`, `Revised="X"`, and
range `[2,2)`.

## Project Structure

The feature will use one public-capability file pair at the repository root:

```text
textprocessor/
├── trim.go
├── trim_test.go
├── match.go
├── segment.go
├── block.go
├── textprocessor.go
├── internal/
└── cmd/
```

The root package remains the stable public facade. Generic `utils` or `common`
packages will not be introduced. If this capability later grows into multiple
complex diff algorithms, implementation details may move to `internal/diff`
without changing the root API.

## Testing

Table-driven tests will cover:

- Replacement and deletion.
- Insertion in the middle, at the start, and at the end.
- Default padding and explicit keep-empty behavior.
- Chinese text, emoji, whitespace, and punctuation with rune offsets.
- A one-rune original, complete replacement, and an empty revised string.
- Empty original input, identical inputs, and an invalid strategy.
- Error identity using `errors.Is`.

Every successful case will additionally verify both result contracts:

- `Original` equals the original rune slice at `[Start, End)`.
- Replacing that rune range with `Revised` reconstructs the full revised input.

## Documentation

README documentation will add a focused `Trim Difference` section showing the
default behavior, rune range semantics, the keep-empty option, and the two
sentinel input errors. No CLI behavior is added.
