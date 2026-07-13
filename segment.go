package textspan

import (
	"strings"
	"unicode"
)

const defaultMinChineseChars = 50

// Span is a text fragment and its rune offsets in the original source.
type Span struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type SegmentOptions struct {
	MinChineseChars int `json:"min_chinese_chars"`
}

// Segment splits Chinese-English mixed text into source spans.
func Segment(text string, opts ...SegmentOptions) []Span {
	options := SegmentOptions{MinChineseChars: defaultMinChineseChars}
	if len(opts) > 0 {
		options = opts[0]
	}

	runes := []rune(text)
	lines := splitLineSpans(runes)
	segments := make([]Span, 0, len(lines))
	for _, line := range lines {
		segments = append(segments, splitLineSegments(runes, line, options.MinChineseChars)...)
	}
	return segments
}

type lineSpan struct {
	start int
	end   int
}

func splitLineSpans(runes []rune) []lineSpan {
	spans := make([]lineSpan, 0)
	start := 0
	for i, r := range runes {
		if r != '\n' {
			continue
		}
		if !isAllWhitespace(runes[start:i]) {
			spans = append(spans, lineSpan{start: start, end: i})
		}
		start = i + 1
	}
	if start < len(runes) && !isAllWhitespace(runes[start:]) {
		spans = append(spans, lineSpan{start: start, end: len(runes)})
	}
	return spans
}

func splitLineSegments(runes []rune, line lineSpan, minChineseChars int) []Span {
	lineText := string(runes[line.start:line.end])
	sentences := Sentences(lineText)
	if len(sentences) == 0 {
		return []Span{{Text: lineText, Start: line.start, End: line.end}}
	}

	rawSegments := make([]Span, 0, len(sentences))
	searchStart := line.start
	for _, sentence := range sentences {
		if sentence == "" {
			continue
		}
		segment, ok := findSentenceSpan(runes, sentence, searchStart, line.end)
		if !ok {
			continue
		}
		rawSegments = append(rawSegments, segment)
		searchStart = segment.End
	}
	if len(rawSegments) == 0 {
		return []Span{{Text: lineText, Start: line.start, End: line.end}}
	}

	rawSegments = expandSegmentsToLine(rawSegments, runes, line)
	return mergeShortSegments(rawSegments, runes, minChineseChars)
}

func expandSegmentsToLine(segments []Span, runes []rune, line lineSpan) []Span {
	if len(segments) == 0 {
		return nil
	}

	expanded := make([]Span, len(segments))
	copy(expanded, segments)
	expanded[0].Start = line.start
	for i := 0; i < len(expanded)-1; i++ {
		expanded[i].End = expanded[i+1].Start
	}
	expanded[len(expanded)-1].End = line.end

	for i := range expanded {
		expanded[i].Text = string(runes[expanded[i].Start:expanded[i].End])
	}
	return expanded
}

func findSentenceSpan(runes []rune, sentence string, start, end int) (Span, bool) {
	target := []rune(sentence)
	if len(target) == 0 || start >= end || len(target) > end-start {
		return Span{}, false
	}

	for i := start; i <= end-len(target); i++ {
		if !hasRuneFragmentAt(runes, target, i) {
			continue
		}
		return Span{Text: string(runes[i : i+len(target)]), Start: i, End: i + len(target)}, true
	}
	return Span{}, false
}

func mergeShortSegments(segments []Span, runes []rune, minChineseChars int) []Span {
	if minChineseChars <= 0 || len(segments) <= 1 {
		return segments
	}

	merged := make([]Span, 0, len(segments))
	for i := 0; i < len(segments); i++ {
		current := segments[i]
		for countChineseRunes(current.Text) < minChineseChars && i+1 < len(segments) {
			i++
			next := segments[i]
			current = Span{
				Text:  string(runes[current.Start:next.End]),
				Start: current.Start,
				End:   next.End,
			}
		}
		merged = append(merged, current)
	}
	return merged
}

func countChineseRunes(s string) int {
	count := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			count++
		}
	}
	return count
}

func hasRuneFragmentAt(runes, fragment []rune, start int) bool {
	if start < 0 || start+len(fragment) > len(runes) {
		return false
	}
	for i, r := range fragment {
		if runes[start+i] != r {
			return false
		}
	}
	return true
}

func isAllWhitespace(runes []rune) bool {
	if len(runes) == 0 {
		return true
	}
	return strings.TrimSpace(string(runes)) == ""
}
