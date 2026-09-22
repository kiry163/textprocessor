package textprocessor

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// ReplaceSpan is a source range to replace together with its replacement text.
// Start and End are half-open rune offsets in the source handed to
// ReplaceSpans, Text is copied verbatim into the result, and ID is a
// caller-supplied identifier that ReplaceSpans passes through unchanged.
type ReplaceSpan[ID comparable] struct {
	ID    ID     `json:"id"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}

// ErrInvalidReplaceSpan indicates a source that is not valid UTF-8, an
// out-of-range replacement range, or overlapping replacement ranges.
var ErrInvalidReplaceSpan = errors.New("invalid replace span")

// ReplaceSpans replaces every part range with its Text and rewrites Start and
// End to the rune offsets the replacement occupies in the returned text.
//
// Ranges are half-open [Start, End): an empty range inserts Text without
// removing source text, and an empty Text deletes the range. Uncovered source
// text, including the head and tail, is copied verbatim.
//
// Parts are sorted in place by Start and End, so the returned slice shares the
// caller's backing array and holds the same elements in ascending order. Every
// returned part satisfies Text == string([]rune(returned)[Start:End]), which
// makes the result directly usable as input for a following replacement pass.
//
// The call is all-or-nothing for coordinates: on error the caller's Start and
// End values are untouched, although an overlap error may already have sorted
// the slice.
func ReplaceSpans[ID comparable](source string, parts []ReplaceSpan[ID]) (string, []ReplaceSpan[ID], error) {
	if !utf8.ValidString(source) {
		return "", parts, fmt.Errorf("%w: source must be valid UTF-8", ErrInvalidReplaceSpan)
	}
	if len(parts) == 0 {
		return source, parts, nil
	}

	sourceRunes := []rune(source)
	for _, part := range parts {
		if part.Start < 0 || part.Start > part.End || part.End > len(sourceRunes) {
			return "", parts, fmt.Errorf(
				"%w: range [%d,%d) is not a valid range of %d runes",
				ErrInvalidReplaceSpan, part.Start, part.End, len(sourceRunes),
			)
		}
	}

	sort.SliceStable(parts, func(i, j int) bool {
		if parts[i].Start != parts[j].Start {
			return parts[i].Start < parts[j].Start
		}
		return parts[i].End < parts[j].End
	})

	size := len(source)
	for i, part := range parts {
		if i > 0 && part.Start < parts[i-1].End {
			return "", parts, fmt.Errorf(
				"%w: range [%d,%d) overlaps [%d,%d)",
				ErrInvalidReplaceSpan, part.Start, part.End, parts[i-1].Start, parts[i-1].End,
			)
		}
		for _, r := range sourceRunes[part.Start:part.End] {
			size -= utf8.RuneLen(r)
		}
		size += len(part.Text)
	}

	var builder strings.Builder
	builder.Grow(size)

	cursor, byteOffset, emitted := 0, 0, 0
	written, replaced := 0, 0
	for i := range parts {
		start, end := parts[i].Start, parts[i].End
		for cursor < start {
			byteOffset += utf8.RuneLen(sourceRunes[cursor])
			cursor++
		}
		builder.WriteString(source[emitted:byteOffset])

		written += start - replaced
		parts[i].Start = written
		builder.WriteString(parts[i].Text)
		written += utf8.RuneCountInString(parts[i].Text)
		parts[i].End = written

		for cursor < end {
			byteOffset += utf8.RuneLen(sourceRunes[cursor])
			cursor++
		}
		emitted = byteOffset
		replaced = end
	}
	builder.WriteString(source[emitted:])

	return builder.String(), parts, nil
}
