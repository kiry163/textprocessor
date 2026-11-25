package textprocessor

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kiry163/ahocorasick"
)

// UniqueSpan is a source span with a substring that occurs once in source.
type UniqueSpan struct {
	Text   string `json:"text"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Unique string `json:"unique"`
}

// ErrInvalidSpan indicates an empty, out-of-range, or inconsistent source span.
var ErrInvalidSpan = errors.New("invalid span")

type uniqueSpanState struct {
	start      int
	end        int
	expandLeft bool
	resolved   bool
}

type uniqueSpanCandidate struct {
	spanIndex    int
	patternIndex int
}

// UniqueSpanOptions controls how BuildUniqueSpansWithOptions decides and
// reports unique spans.
type UniqueSpanOptions struct {
	// IgnoreWhitespace decides uniqueness after removing Unicode whitespace
	// from both the source and every candidate, so a newline or space alone
	// never makes a span unique. Unique stays an exact source substring, and
	// whitespace added around the input span is trimmed.
	IgnoreWhitespace bool

	// MaxRunes caps Unique in runes. Zero means no cap. The cap is raised to
	// the input span length when the span itself is longer. If the cap is
	// reached without a unique candidate, the window at the cap is returned
	// without a uniqueness guarantee.
	MaxRunes int
}

// BuildUniqueSpans adds an exact source substring that uniquely identifies
// every input span. Non-unique spans expand one rune at a time, alternating
// left and right and falling back to the other side at a source boundary.
func BuildUniqueSpans(source string, spans []Span) ([]UniqueSpan, error) {
	return BuildUniqueSpansWithOptions(source, spans, UniqueSpanOptions{})
}

// BuildUniqueSpansWithOptions is BuildUniqueSpans with configurable uniqueness
// and length rules.
func BuildUniqueSpansWithOptions(source string, spans []Span, opts UniqueSpanOptions) ([]UniqueSpan, error) {
	if !utf8.ValidString(source) {
		return nil, fmt.Errorf("%w: source must be valid UTF-8", ErrInvalidSpan)
	}
	sourceRunes := []rune(source)
	countSource := source
	ignoreWhitespace := opts.IgnoreWhitespace
	if ignoreWhitespace {
		if normalized := removeWhitespace(source); normalized != "" {
			countSource = normalized
		} else {
			// A source made only of whitespace has no content to compare;
			// fall back to exact matching so the span can still resolve.
			ignoreWhitespace = false
		}
	}
	results := make([]UniqueSpan, len(spans))
	states := make([]uniqueSpanState, len(spans))
	limits := make([]int, len(spans))

	for i, span := range spans {
		if err := validateSourceSpan(sourceRunes, span); err != nil {
			return nil, fmt.Errorf("%w at index %d: %v", ErrInvalidSpan, i, err)
		}
		results[i] = UniqueSpan{Text: span.Text, Start: span.Start, End: span.End}
		states[i] = uniqueSpanState{start: span.Start, end: span.End, expandLeft: true}
		limits[i] = uniqueSpanLimit(span, opts.MaxRunes)
	}

	unresolved := len(spans)
	firstRound := true
	for unresolved > 0 {
		patterns := make([]string, 0, unresolved)
		patternIndexes := make(map[string]int, unresolved)
		candidates := make([]uniqueSpanCandidate, 0, unresolved)

		for i := range states {
			state := &states[i]
			if state.resolved {
				continue
			}
			if !firstRound && !expandUniqueSpan(state, len(sourceRunes)) {
				return nil, fmt.Errorf("%w at index %d: cannot expand range", ErrInvalidSpan, i)
			}

			candidateText := string(sourceRunes[state.start:state.end])
			patternText := candidateText
			if ignoreWhitespace {
				// A candidate made only of whitespace has no content to compare;
				// keep it exact so it never resolves through the stripped source.
				if normalized := removeWhitespace(candidateText); normalized != "" {
					patternText = normalized
				}
			}
			patternIndex, ok := patternIndexes[patternText]
			if !ok {
				patternIndex = len(patterns)
				patternIndexes[patternText] = patternIndex
				patterns = append(patterns, patternText)
			}
			candidates = append(candidates, uniqueSpanCandidate{
				spanIndex:    i,
				patternIndex: patternIndex,
			})
		}

		counts, err := countOverlappingPatterns(countSource, patterns)
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			if counts[candidate.patternIndex] != 1 {
				continue
			}
			state := &states[candidate.spanIndex]
			results[candidate.spanIndex].Unique = uniqueSpanText(sourceRunes, spans[candidate.spanIndex], state.start, state.end, ignoreWhitespace)
			state.resolved = true
			unresolved--
		}
		if opts.MaxRunes > 0 {
			for i := range states {
				state := &states[i]
				if state.resolved || state.end-state.start < limits[i] {
					continue
				}
				results[i].Unique = uniqueSpanText(sourceRunes, spans[i], state.start, state.end, ignoreWhitespace)
				state.resolved = true
				unresolved--
			}
		}
		firstRound = false
	}

	return results, nil
}

func validateSourceSpan(sourceRunes []rune, span Span) error {
	if span.Start < 0 || span.Start >= span.End || span.End > len(sourceRunes) {
		return fmt.Errorf("range [%d,%d) is outside source rune length %d or empty", span.Start, span.End, len(sourceRunes))
	}
	fragment := string(sourceRunes[span.Start:span.End])
	if span.Text != fragment {
		return fmt.Errorf("text %q does not equal source fragment %q at [%d,%d)", span.Text, fragment, span.Start, span.End)
	}
	return nil
}

func expandUniqueSpan(state *uniqueSpanState, sourceLength int) bool {
	if state.start == 0 && state.end == sourceLength {
		return false
	}

	if state.expandLeft {
		if state.start > 0 {
			state.start--
		} else {
			state.end++
		}
	} else {
		if state.end < sourceLength {
			state.end++
		} else {
			state.start--
		}
	}
	state.expandLeft = !state.expandLeft
	return true
}

func countOverlappingPatterns(source string, patterns []string) ([]int, error) {
	matcher, err := ahocorasick.Compile(patterns, ahocorasick.Options{})
	if err != nil {
		return nil, fmt.Errorf("build unique span matcher: %w", err)
	}

	counts := make([]int, len(patterns))
	for _, match := range matcher.FindAllOverlapping(source) {
		counts[match.Pattern()]++
	}
	return counts, nil
}

// removeWhitespace deletes every Unicode whitespace rune. It is used both for
// the counted source and for candidates so that a newline or space alone cannot
// distinguish two occurrences.
func removeWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// uniqueSpanLimit returns the rune cap for one span. Zero means uncapped; a
// cap never shrinks the input span itself.
func uniqueSpanLimit(span Span, maxRunes int) int {
	if maxRunes <= 0 {
		return 0
	}
	if length := span.End - span.Start; length > maxRunes {
		return length
	}
	return maxRunes
}

// uniqueSpanText returns the window and, when whitespace is ignored, trims
// whitespace added around the original span without touching the span itself.
func uniqueSpanText(sourceRunes []rune, span Span, start, end int, ignoreWhitespace bool) string {
	if ignoreWhitespace {
		for start < span.Start && unicode.IsSpace(sourceRunes[start]) {
			start++
		}
		for end > span.End && unicode.IsSpace(sourceRunes[end-1]) {
			end--
		}
	}
	return string(sourceRunes[start:end])
}
