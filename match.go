package textspan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type MatchMode string

const (
	MatchExact      MatchMode = "exact"
	MatchNormalized MatchMode = "normalized"
	MatchSmart      MatchMode = "smart"
)

type MatchBoundary string

const (
	MatchBoundaryNone       MatchBoundary = "none"
	MatchBoundaryQueryEdges MatchBoundary = "query_edges"
)

type MatchOptions struct {
	Mode     MatchMode     `json:"mode"`
	Boundary MatchBoundary `json:"boundary"`
}

// MatchResult is a fragment found in source text with rune offsets.
type MatchResult struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type normalizedUnit struct {
	normStart int
	normEnd   int
	srcStart  int
	srcEnd    int
}

// Match locates query in source with the requested matching mode.
func Match(source, query string, opts ...MatchOptions) []MatchResult {
	options := MatchOptions{Mode: MatchSmart, Boundary: MatchBoundaryQueryEdges}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.Mode == "" {
		options.Mode = MatchSmart
	}
	if options.Boundary == "" {
		options.Boundary = MatchBoundaryQueryEdges
	}

	switch options.Mode {
	case MatchExact:
		return exactMatches(source, query)
	case MatchNormalized:
		return normalizedMatches(source, query, options.Boundary)
	case MatchSmart:
		matches := exactMatches(source, query)
		if len(matches) > 0 {
			return matches
		}
		return normalizedMatches(source, query, options.Boundary)
	default:
		return nil
	}
}

func exactMatches(source, query string) []MatchResult {
	if query == "" {
		return nil
	}

	var matches []MatchResult
	offset := 0
	for {
		idx := strings.Index(source[offset:], query)
		if idx < 0 {
			return matches
		}
		startByte := offset + idx
		endByte := startByte + len(query)
		matches = append(matches, MatchResult{
			Text:  source[startByte:endByte],
			Start: runeOffset(source[:startByte]),
			End:   runeOffset(source[:endByte]),
		})
		offset = endByte
	}
}

func normalizedMatches(source, query string, boundary MatchBoundary) []MatchResult {
	normalizedNeedle, _ := normalizeTextWithUnits(query)
	if normalizedNeedle == "" {
		return nil
	}

	normalizedSource, units := normalizeTextWithUnits(source)
	if normalizedSource == "" {
		return nil
	}

	var matches []MatchResult
	offset := 0
	for {
		idx := strings.Index(normalizedSource[offset:], normalizedNeedle)
		if idx < 0 {
			return matches
		}

		start := offset + idx
		end := start + len(normalizedNeedle)
		srcStart, srcEnd, ok := sourceSpanForNormalizedMatch(source, query, units, start, end, boundary)
		if ok && srcStart >= 0 && srcEnd <= len(source) && srcStart < srcEnd {
			matches = append(matches, MatchResult{
				Text:  source[srcStart:srcEnd],
				Start: runeOffset(source[:srcStart]),
				End:   runeOffset(source[:srcEnd]),
			})
		}
		offset = end
	}
}

func runeOffset(text string) int {
	return utf8.RuneCountInString(text)
}

func normalizeTextWithUnits(text string) (string, []normalizedUnit) {
	var builder strings.Builder
	units := []normalizedUnit{}
	for srcStart, r := range text {
		if shouldDropForMatch(r) {
			continue
		}

		srcEnd := srcStart + runeByteWidth(r)
		normStart := builder.Len()
		builder.WriteRune(r)
		units = append(units, normalizedUnit{
			normStart: normStart,
			normEnd:   builder.Len(),
			srcStart:  srcStart,
			srcEnd:    srcEnd,
		})
	}
	return builder.String(), units
}

func shouldDropForMatch(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.Is(unicode.Cf, r)
}

func runeByteWidth(r rune) int {
	width := utf8.RuneLen(r)
	if width < 0 {
		return 1
	}
	return width
}

func sourceSpanForNormalizedMatch(source, fragment string, units []normalizedUnit, normStart, normEnd int, boundary MatchBoundary) (int, int, bool) {
	srcStart := -1
	srcEnd := -1
	for _, unit := range units {
		if unit.normStart == normStart {
			srcStart = unit.srcStart
		}
		if unit.normEnd == normEnd {
			srcEnd = unit.srcEnd
			break
		}
	}
	if srcStart < 0 || srcEnd < 0 {
		return srcStart, srcEnd, false
	}

	if boundary == MatchBoundaryQueryEdges {
		fragmentLeft, fragmentRight := droppedEdges(fragment)
		srcStart = recoverLeftQueryEdge(source, srcStart, len([]rune(fragmentLeft)))
		srcEnd = recoverRightQueryEdge(source, srcEnd, len([]rune(fragmentRight)))
	}
	return srcStart, srcEnd, true
}

func recoverLeftQueryEdge(source string, boundary, maxRunes int) int {
	for maxRunes > 0 && boundary > 0 {
		r, width := utf8.DecodeLastRuneInString(source[:boundary])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		boundary -= width
		maxRunes--
	}
	return boundary
}

func recoverRightQueryEdge(source string, boundary, maxRunes int) int {
	for maxRunes > 0 && boundary < len(source) {
		r, width := utf8.DecodeRuneInString(source[boundary:])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		boundary += width
		maxRunes--
	}
	return boundary
}

func droppedEdges(text string) (left, right string) {
	start := 0
	for start < len(text) {
		r, width := utf8.DecodeRuneInString(text[start:])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		start += width
	}

	end := len(text)
	for end > start {
		r, width := utf8.DecodeLastRuneInString(text[:end])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		end -= width
	}

	return text[:start], text[end:]
}

func droppedRunesLeft(source string, boundary int) (int, string) {
	start := boundary
	for start > 0 {
		r, width := utf8.DecodeLastRuneInString(source[:start])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		start -= width
	}
	return start, source[start:boundary]
}

func droppedRunesRight(source string, boundary int) (int, string) {
	end := boundary
	for end < len(source) {
		r, width := utf8.DecodeRuneInString(source[end:])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !shouldDropForMatch(r) {
			break
		}
		end += width
	}
	return end, source[boundary:end]
}

func commonBoundarySuffix(sourceEdge, fragmentEdge string) string {
	sourceRunes := []rune(sourceEdge)
	fragmentRunes := []rune(fragmentEdge)
	matched := 0
	for matched < len(sourceRunes) && matched < len(fragmentRunes) {
		if sourceRunes[len(sourceRunes)-1-matched] != fragmentRunes[len(fragmentRunes)-1-matched] {
			break
		}
		matched++
	}
	if matched == 0 {
		return ""
	}
	return string(sourceRunes[len(sourceRunes)-matched:])
}

func commonBoundaryPrefix(sourceEdge, fragmentEdge string) string {
	sourceRunes := []rune(sourceEdge)
	fragmentRunes := []rune(fragmentEdge)
	matched := 0
	for matched < len(sourceRunes) && matched < len(fragmentRunes) {
		if sourceRunes[matched] != fragmentRunes[matched] {
			break
		}
		matched++
	}
	if matched == 0 {
		return ""
	}
	return string(sourceRunes[:matched])
}
