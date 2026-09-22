package textprocessor

import (
	"strings"
	"unicode/utf8"
)

// SentenceAt returns the sentence that contains the rune at pos, as a tight
// span in the original text. The span is half-open, so pos belongs to the
// sentence when Start <= pos < End.
//
// The returned span is not merged with neighbouring short spans and is not
// padded to the surrounding line. It reports false when pos is out of range or
// falls outside every sentence, such as on a newline, on a blank line, on a
// line indentation, or on whitespace between sentences.
//
// Only the position's own line is segmented. The call takes O(pos) time to
// resolve the rune offset and O(line) additional memory, and it does not copy
// the whole text. Invalid UTF-8 bytes are replaced with U+FFFD, matching the
// []rune conversion used by Sentences and Segment.
func SentenceAt(text string, pos int) (Span, bool) {
	bytePos, ok := byteOffsetAtRune(text, pos)
	if !ok {
		return Span{}, false
	}

	lineStart := strings.LastIndexByte(text[:bytePos], '\n') + 1
	lineEnd := len(text)
	if nextBreak := strings.IndexByte(text[bytePos:], '\n'); nextBreak >= 0 {
		lineEnd = bytePos + nextBreak
	}
	lineRunes := []rune(text[lineStart:lineEnd])
	lineText := string(lineRunes)
	if strings.TrimSpace(lineText) == "" {
		return Span{}, false
	}

	base := utf8.RuneCountInString(text[:lineStart])
	posInLine := pos - base
	searchStart := 0
	for _, sentence := range Sentences(lineText) {
		if sentence == "" {
			continue
		}
		span, found := findSentenceSpan(lineRunes, sentence, searchStart, len(lineRunes))
		if !found {
			continue
		}
		searchStart = span.End
		if span.Start > posInLine {
			break
		}
		if posInLine < span.End {
			return Span{Text: span.Text, Start: base + span.Start, End: base + span.End}, true
		}
	}
	return Span{}, false
}

// byteOffsetAtRune returns the byte offset of the rune at index pos. Invalid
// UTF-8 bytes count as one rune each, matching []rune conversion.
func byteOffsetAtRune(text string, pos int) (int, bool) {
	if pos < 0 {
		return 0, false
	}

	count := 0
	for i := range text {
		if count == pos {
			return i, true
		}
		count++
	}
	return 0, false
}
