package textprocessor

import (
	"errors"
	"fmt"

	"github.com/kiry163/ahocorasick"
)

// BatchMatchOptions controls multi-pattern matching semantics.
type BatchMatchOptions struct {
	Overlapping      bool `json:"overlapping"`
	ASCIIInsensitive bool `json:"ascii_insensitive"`
	WholeWords       bool `json:"whole_words"`
}

// BatchMatchResult identifies a query and its half-open rune range in source.
type BatchMatchResult struct {
	Query string `json:"query"`
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// ErrInvalidBatchQueries indicates that queries cannot be compiled for matching.
var ErrInvalidBatchQueries = errors.New("invalid batch queries")

// MatchBatch locates multiple queries in source with an Aho-Corasick matcher.
// Duplicate queries are treated as one pattern. By default, matches are
// non-overlapping and use leftmost-longest selection.
func MatchBatch(source string, queries []string, opts ...BatchMatchOptions) ([]BatchMatchResult, error) {
	patterns := uniqueStrings(queries)
	options := BatchMatchOptions{}
	if len(opts) > 0 {
		options = opts[0]
	}

	matcher, err := ahocorasick.Compile(patterns, ahocorasick.Options{
		ASCIIInsensitive: options.ASCIIInsensitive,
		WholeWords:       options.WholeWords,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBatchQueries, err)
	}

	var matches []ahocorasick.Match
	if options.Overlapping {
		matches = matcher.FindAllOverlapping(source)
	} else {
		matches = matcher.FindAll(source)
	}

	results := make([]BatchMatchResult, 0, len(matches))
	for _, match := range matches {
		results = append(results, BatchMatchResult{
			Query: patterns[match.Pattern()],
			Text:  source[match.ByteStart():match.ByteEnd()],
			Start: match.Start(),
			End:   match.End(),
		})
	}
	return results, nil
}

func uniqueStrings(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
