# textprocessor

`textprocessor` is a Go library for Chinese-English mixed text segmentation and
source-span matching.

## Install

```bash
go get github.com/kiry163/textprocessor
```

## Sentence Segmentation

```go
sentences := textprocessor.Sentences("这是第一句。This is sentence two.")
// []string{"这是第一句。", "This is sentence two."}
```

The default sentence rules are tuned for Chinese-English mixed text. There is
no language selector in the public API.

## Sentence At Position

```go
span, ok := textprocessor.SentenceAt("你好。Hello.", 5)
// span == textprocessor.Span{Text: "Hello.", Start: 3, End: 9}
// ok == true
```

`SentenceAt` returns the sentence containing one rune position. `pos` is a rune
index and the result is a half-open `[Start, End)` range, so `pos` belongs to
the sentence when `Start <= pos < End`: the first rune after a sentence belongs
to the next one. Sentences are resolved per line, so a sentence never crosses a
newline.

The returned span is tight. It is neither merged with neighbouring short
Chinese spans nor padded to the line edges, so it can be shorter than the span
covering the same runes in `Segment`:

```go
text := "  缩进 你好。 尾随 "
textprocessor.Segment(text, textprocessor.SegmentOptions{MinChineseChars: 0})
// []textprocessor.Span{
//     {Text: "  缩进 你好。 ", Start: 0, End: 9},
//     {Text: "尾随 ", Start: 9, End: 12},
// }
```

| Position | `SentenceAt` | covering `Segment` span |
| --- | --- | --- |
| `0`, `1` | not found (indentation) | `[0,9)` |
| `2`–`7` | `[2,8)` `缩进 你好。` | `[0,9)` |
| `8` | not found (space between sentences) | `[0,9)` |
| `9`, `10` | `[9,11)` `尾随` | `[9,12)` |
| `11`, `12` | not found (trailing space, end of text) | `[9,12)` |

`ok` is `false` for every position that no sentence contains: negative or
out-of-range positions, newline runes, blank lines, line indentation, and
whitespace between or after sentences. The zero `Span` is returned alongside
`false`.

## Text Spans

```go
spans := textprocessor.Segment("你好。Hello.", textprocessor.SegmentOptions{
    MinChineseChars: 0,
})
// []textprocessor.Span{
//     {Text: "你好。", Start: 0, End: 3},
//     {Text: "Hello.", Start: 3, End: 9},
// }
```

`Start` and `End` are rune offsets in the original input. By default, short
Chinese spans are merged until they contain at least 50 Han characters. Use
`SegmentOptions{MinChineseChars: 0}` to disable this merging.

## Document Blocks

```go
blocks := textprocessor.Blocks(markdownText)
```

`Blocks` splits Markdown-like text into structural blocks and returns rune
offsets in the original input. By default, each non-empty ordinary line is a
separate paragraph block. Use `ParagraphSplitByBlankLine` for Markdown-style
blank-line paragraph grouping:

```go
blocks := textprocessor.Blocks(markdownText, textprocessor.BlockOptions{
    ParagraphSplit: textprocessor.ParagraphSplitByBlankLine,
})
```

`Blocks` emits:

```go
textprocessor.BlockParagraph
textprocessor.BlockTable
textprocessor.BlockFormula
textprocessor.BlockCode
```

Markdown pipe tables are recognized only when rows start and end with `|` and
the second row is a standard separator row. Independent LaTeX formulas are
recognized for `$$...$$`, `\[...\]`, and
`\begin{equation}...\end{equation}` blocks. Fenced Markdown code blocks using
``` or `~~~` are emitted as `BlockCode`.

## Source Matching

```go
matches := textprocessor.Match("他说：Hello, world！", "Hello world")
// []textprocessor.MatchResult{{Text: "Hello, world", Start: 3, End: 15}}
```

The default mode is `MatchSmart`: it returns exact matches first, and only
falls back to normalized matching when no exact match exists.

Available modes:

```go
textprocessor.MatchExact
textprocessor.MatchNormalized
textprocessor.MatchSmart
```

`MatchExact` only searches exact source text. `MatchNormalized` ignores
whitespace, punctuation, and Unicode format characters. `MatchSmart` prefers
exact matches and falls back to normalized matches.

Normalized matching recovers ignored query-edge characters by default. For
example, when the query is `封盖。` and the source contains `封盖，`, the match
result includes the source-side comma: `封盖，`. Disable this boundary recovery
when needed:

```go
matches := textprocessor.Match(source, "封盖。", textprocessor.MatchOptions{
    Mode:     textprocessor.MatchNormalized,
    Boundary: textprocessor.MatchBoundaryNone,
})
```

## Batch Matching

`MatchBatch` searches for multiple exact strings with an Aho-Corasick matcher:

```go
matches, err := textprocessor.MatchBatch(
    "Go 项目和 Go",
    []string{"Go", "Go 项目", "Go"},
)
// []textprocessor.BatchMatchResult{
//     {Query: "Go 项目", Text: "Go 项目", Start: 0, End: 5},
//     {Query: "Go", Text: "Go", Start: 7, End: 9},
// }
```

Duplicate queries are treated as one pattern. The default selection is
non-overlapping leftmost-longest matching. Set `Overlapping` to include every
matching pattern, including shorter patterns at the same source position:

```go
matches, err := textprocessor.MatchBatch(source, queries,
    textprocessor.BatchMatchOptions{Overlapping: true},
)
```

`BatchMatchOptions` also supports `ASCIIInsensitive` and `WholeWords`. `Query`
is the compiled query string, while `Text` is the actual source substring. As
with the other span APIs, `Start` and `End` are half-open rune offsets.

Invalid query sets, including empty query lists and empty or invalid UTF-8
queries, return an error matching `ErrInvalidBatchQueries`.

## Unique Source Spans

`BuildUniqueSpans` adds a substring that occurs exactly once in the source:

```go
uniqueSpans, err := textprocessor.BuildUniqueSpans(source, []textprocessor.Span{
    {Text: "目标", Start: 2, End: 4},
})
// []textprocessor.UniqueSpan{
//     {Text: "目标", Start: 2, End: 4, Unique: "目标"},
// }
```

If `Text` already occurs once, `Unique` is unchanged. Otherwise, unresolved
spans are matched in batches and expanded cumulatively by one rune, alternating
left then right. At a source boundary, expansion automatically continues on
the other side. `Unique` is the first substring on that expansion path that
occurs once; it is not guaranteed to be the shortest possible unique
substring. Occurrence counting includes overlapping matches.

The source must be valid UTF-8. Every input must be a non-empty, valid source
range and must satisfy:

```go
span.Text == string([]rune(source)[span.Start:span.End])
```

An invalid range or inconsistent text returns an error matching
`ErrInvalidSpan`.

## Trim Difference

`TrimDifference` removes the exact common rune prefix and suffix from an
original and revised string, then reports the changed range in the original:

```go
result, err := textprocessor.TrimDifference("abc", "abXc")
// result == textprocessor.TrimResult{
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
result, err := textprocessor.TrimDifference("abc", "abXc", textprocessor.TrimOptions{
    EmptyOriginal: textprocessor.TrimEmptyOriginalKeep,
})
// result == textprocessor.TrimResult{
//     Original: "",
//     Revised:  "X",
//     Start:    2,
//     End:      2,
// }
```

An empty original returns `ErrOriginalEmpty`. Identical inputs return
`ErrNoChange`. Invalid strategy values return an error matching
`ErrInvalidTrimOptions`; use `errors.Is` when branching on these errors.

## Span Replacement

`ReplaceSpans` replaces source ranges with new text and rewrites those ranges to
the rune offsets the replacements occupy in the result:

```go
text, spans, err := textprocessor.ReplaceSpans("你好。Hello.", []textprocessor.ReplaceSpan[string]{
    {ID: "s2", Start: 3, End: 9, Text: "Hello!"},
    {ID: "s1", Start: 0, End: 3, Text: "你好！"},
})
// text  == "你好！Hello!"
// spans == []textprocessor.ReplaceSpan[string]{
//     {ID: "s1", Start: 0, End: 3, Text: "你好！"},
//     {ID: "s2", Start: 3, End: 9, Text: "Hello!"},
// }
```

`Start` and `End` are half-open rune offsets in the source, and `Text` is copied
verbatim into the result. An empty range inserts text and an empty `Text`
deletes the range; source text that no span covers, including the head and the
tail, is kept unchanged. `ID` is caller-defined and returned unchanged, and the
type parameter preserves its Go type.

Parts are sorted in place by `Start` and `End`, so the returned slice shares the
caller's backing array and holds the same elements in ascending order. Every
returned part satisfies `Text == string([]rune(returned)[Start:End])`, which
makes the result directly usable as the input of the next replacement pass:

```go
text, spans, err = textprocessor.ReplaceSpans(text, spans)
```

Ranges must not overlap: `[0,3)` and `[3,5)` may coexist, and an empty range at
a boundary of another range is not an overlap, while `[0,3)` and `[2,5)` are.
Invalid UTF-8 sources, out-of-range ranges, and overlapping ranges return an
error matching `ErrInvalidReplaceSpan`. The call is all-or-nothing for
coordinates: on error the caller's `Start` and `End` values are untouched,
although an overlap error may already have sorted the slice.

## Full Text Diff

`Diff` returns the complete equal, inserted, and deleted sequence between two
texts:

```go
parts, err := textprocessor.Diff("我喜欢苹果和香蕉", "我喜欢红苹果和葡萄")
```

Every `DiffPart` contains half-open rune ranges in both the original and
revised text. Insertions have an empty original range, while deletions have an
empty revised range. Concatenating equal and deleted parts reconstructs the
original; concatenating equal and inserted parts reconstructs the revision.

Semantic cleanup is enabled by default to favor readable edit boundaries. Use
`DiffCleanupNone` when the engine's merged result is preferred:

```go
parts, err := textprocessor.Diff(original, revised, textprocessor.DiffOptions{
    Cleanup: textprocessor.DiffCleanupNone,
})
```

Set `Timeout` to limit expensive comparisons. A zero timeout has no limit.
Inputs must be valid UTF-8. Invalid input and options return errors matching
`ErrInvalidDiffInput` and `ErrInvalidDiffOptions` respectively.

Render the sequence as escaped semantic HTML:

```go
fragment, err := textprocessor.RenderDiffHTML(parts)
```

The fragment uses a `<pre>` container together with `<span>`, `<ins>`, and
`<del>` elements. `DefaultDiffHTMLCSS` provides ready-to-use styles. CSS class
names can be replaced without changing the generated diff:

```go
fragment, err := textprocessor.RenderDiffHTML(parts, textprocessor.DiffHTMLStyle{
    ContainerClass: "document-diff",
    EqualClass:     "unchanged",
    InsertClass:    "added",
    DeleteClass:    "removed",
})
```

`RenderDiffText` uses ANSI green and red by default. Supply a `DiffTextStyle`
to use custom markers or omit unchanged text.

To generate a complete standalone HTML document with UTF-8 metadata and a
stylesheet, use `RenderDiffHTMLDocument`:

```go
document, err := textprocessor.RenderDiffHTMLDocument(parts,
    textprocessor.DiffHTMLDocumentOptions{
        Title: "Review",
        Language: "zh-CN",
        Theme: textprocessor.DiffHTMLThemeDark,
    },
)
```

Built-in themes are `DiffHTMLThemeLight`, `DiffHTMLThemeDark`, and
`DiffHTMLThemeHighContrast`. `DiffHTMLThemeCSS` returns a theme stylesheet for
embedding elsewhere. Add trusted CSS through `AdditionalCSS` when a theme
needs application-specific overrides. HTML output contains only semantic
elements and classes; it does not include source character offsets.

## CLI

```bash
go run ./cmd/textprocessor input.txt
```

The default CLI output is JSON Lines:

```json
{"text":"第一句。","start":0,"end":4}
{"text":"Second sentence.","start":4,"end":20}
```

Useful flags:

```bash
go run ./cmd/textprocessor --sentences input.txt
go run ./cmd/textprocessor --min-chinese-chars 0 input.txt
```

Split document blocks:

```bash
go run ./cmd/textprocessor blocks input.md
go run ./cmd/textprocessor blocks --paragraph-split blank-line input.md
```

Match a query string against a source file:

```bash
go run ./cmd/textprocessor match --mode smart source.txt "Hello world"
go run ./cmd/textprocessor match --mode exact source.txt "Hello world"
go run ./cmd/textprocessor match --mode normalized source.txt "Hello world"
```

Quote the query when it contains spaces or shell-sensitive characters.

Match output is also JSON Lines:

```json
{"text":"Hello, world","start":3,"end":15}
```
