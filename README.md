# textspan

`textspan` is a Go library for Chinese-English mixed text segmentation and
source-span matching.

## Install

```bash
go get github.com/kiry163/textprocessor
```

## Sentence Segmentation

```go
sentences := textspan.Sentences("这是第一句。This is sentence two.")
// []string{"这是第一句。", "This is sentence two."}
```

The default sentence rules are tuned for Chinese-English mixed text. There is
no language selector in the public API.

## Text Spans

```go
spans := textspan.Segment("你好。Hello.", textspan.SegmentOptions{
    MinChineseChars: 0,
})
// []textspan.Span{
//     {Text: "你好。", Start: 0, End: 3},
//     {Text: "Hello.", Start: 3, End: 9},
// }
```

`Start` and `End` are rune offsets in the original input. By default, short
Chinese spans are merged until they contain at least 50 Han characters. Use
`SegmentOptions{MinChineseChars: 0}` to disable this merging.

## Document Blocks

```go
blocks := textspan.Blocks(markdownText)
```

`Blocks` splits Markdown-like text into structural blocks and returns rune
offsets in the original input. By default, each non-empty ordinary line is a
separate paragraph block. Use `ParagraphSplitByBlankLine` for Markdown-style
blank-line paragraph grouping:

```go
blocks := textspan.Blocks(markdownText, textspan.BlockOptions{
    ParagraphSplit: textspan.ParagraphSplitByBlankLine,
})
```

`Blocks` emits:

```go
textspan.BlockParagraph
textspan.BlockTable
textspan.BlockFormula
textspan.BlockCode
```

Markdown pipe tables are recognized only when rows start and end with `|` and
the second row is a standard separator row. Independent LaTeX formulas are
recognized for `$$...$$`, `\[...\]`, and
`\begin{equation}...\end{equation}` blocks. Fenced Markdown code blocks using
``` or `~~~` are emitted as `BlockCode`.

## Source Matching

```go
matches := textspan.Match("他说：Hello, world！", "Hello world")
// []textspan.MatchResult{{Text: "Hello, world", Start: 3, End: 15}}
```

The default mode is `MatchSmart`: it returns exact matches first, and only
falls back to normalized matching when no exact match exists.

Available modes:

```go
textspan.MatchExact
textspan.MatchNormalized
textspan.MatchSmart
```

`MatchExact` only searches exact source text. `MatchNormalized` ignores
whitespace, punctuation, and Unicode format characters. `MatchSmart` prefers
exact matches and falls back to normalized matches.

Normalized matching recovers ignored query-edge characters by default. For
example, when the query is `封盖。` and the source contains `封盖，`, the match
result includes the source-side comma: `封盖，`. Disable this boundary recovery
when needed:

```go
matches := textspan.Match(source, "封盖。", textspan.MatchOptions{
    Mode:     textspan.MatchNormalized,
    Boundary: textspan.MatchBoundaryNone,
})
```

## Trim Difference

`TrimDifference` removes the exact common rune prefix and suffix from an
original and revised string, then reports the changed range in the original:

```go
result, err := textspan.TrimDifference("abc", "abXc")
// result == textspan.TrimResult{
//     Original: "b",
//     Revised:  "bX",
//     Start:    1,
//     End:      2,
// }
```

`Start` and `End` are rune offsets in the original input. Replacing that
half-open range `[Start, End)` with `Revised` reconstructs the complete revised
input.

An insertion has an empty minimal original range. By default, the result
includes the shared rune immediately to its left; an insertion at the start
uses the shared rune to its right. This default is `TrimEmptyOriginalPad`.
Preserve the minimal empty range when needed:

```go
result, err := textspan.TrimDifference("abc", "abXc", textspan.TrimOptions{
    EmptyOriginal: textspan.TrimEmptyOriginalKeep,
})
// result == textspan.TrimResult{
//     Original: "",
//     Revised:  "X",
//     Start:    2,
//     End:      2,
// }
```

An empty original returns `ErrOriginalEmpty`. Identical inputs return
`ErrNoChange`. Invalid strategy values return an error matching
`ErrInvalidTrimOptions`; use `errors.Is` when branching on these errors.

## CLI

```bash
go run ./cmd/textspan input.txt
```

The default CLI output is JSON Lines:

```json
{"text":"第一句。","start":0,"end":4}
{"text":"Second sentence.","start":4,"end":20}
```

Useful flags:

```bash
go run ./cmd/textspan --sentences input.txt
go run ./cmd/textspan --min-chinese-chars 0 input.txt
```

Split document blocks:

```bash
go run ./cmd/textspan blocks input.md
go run ./cmd/textspan blocks --paragraph-split blank-line input.md
```

Match a query string against a source file:

```bash
go run ./cmd/textspan match --mode smart source.txt "Hello world"
go run ./cmd/textspan match --mode exact source.txt "Hello world"
go run ./cmd/textspan match --mode normalized source.txt "Hello world"
```

Quote the query when it contains spaces or shell-sensitive characters.

Match output is also JSON Lines:

```json
{"text":"Hello, world","start":3,"end":15}
```
