package textprocessor

import (
	"errors"
	"fmt"
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

// BuildUniqueSpans adds an exact source substring that uniquely identifies
// every input span. Non-unique spans expand one rune at a time, alternating
// left and right and falling back to the other side at a source boundary.
func BuildUniqueSpans(source string, spans []Span) ([]UniqueSpan, error) {
	if !utf8.ValidString(source) {
		return nil, fmt.Errorf("%w: source must be valid UTF-8", ErrInvalidSpan)
	}
	sourceRunes := []rune(source)
	results := make([]UniqueSpan, len(spans))
	states := make([]uniqueSpanState, len(spans))

	for i, span := range spans {
		if err := validateSourceSpan(sourceRunes, span); err != nil {
			return nil, fmt.Errorf("%w at index %d: %v", ErrInvalidSpan, i, err)
		}
		results[i] = UniqueSpan{Text: span.Text, Start: span.Start, End: span.End}
		states[i] = uniqueSpanState{start: span.Start, end: span.End, expandLeft: true}
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
			patternIndex, ok := patternIndexes[candidateText]
			if !ok {
				patternIndex = len(patterns)
				patternIndexes[candidateText] = patternIndex
				patterns = append(patterns, candidateText)
			}
			candidates = append(candidates, uniqueSpanCandidate{
				spanIndex:    i,
				patternIndex: patternIndex,
			})
		}

		counts, err := countOverlappingPatterns(source, patterns)
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			if counts[candidate.patternIndex] != 1 {
				continue
			}
			state := &states[candidate.spanIndex]
			results[candidate.spanIndex].Unique = string(sourceRunes[state.start:state.end])
			state.resolved = true
			unresolved--
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
