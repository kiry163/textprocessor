package textspan

import (
	"errors"
	"fmt"
)

// TrimResult contains the differing text and its rune range in the original.
type TrimResult struct {
	Original string `json:"original"`
	Revised  string `json:"revised"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
}

// TrimEmptyOriginalStrategy controls insertion-only results.
type TrimEmptyOriginalStrategy string

const (
	// TrimEmptyOriginalPad includes one shared context rune in insertion-only results.
	TrimEmptyOriginalPad TrimEmptyOriginalStrategy = "pad"
	// TrimEmptyOriginalKeep preserves the empty original insertion range.
	TrimEmptyOriginalKeep TrimEmptyOriginalStrategy = "keep"
)

// TrimOptions configures TrimDifference.
type TrimOptions struct {
	EmptyOriginal TrimEmptyOriginalStrategy `json:"empty_original"`
}

var (
	// ErrOriginalEmpty indicates that original contains no runes.
	ErrOriginalEmpty = errors.New("original must not be empty")
	// ErrNoChange indicates that original and revised are identical.
	ErrNoChange = errors.New("original and revised have no change")
	// ErrInvalidTrimOptions indicates an unsupported option value.
	ErrInvalidTrimOptions = errors.New("invalid trim options")
)

// TrimDifference removes the exact common rune prefix and suffix from two texts.
func TrimDifference(original, revised string, opts ...TrimOptions) (TrimResult, error) {
	originalRunes := []rune(original)
	revisedRunes := []rune(revised)
	if len(originalRunes) == 0 {
		return TrimResult{}, ErrOriginalEmpty
	}
	if equalRunes(originalRunes, revisedRunes) {
		return TrimResult{}, ErrNoChange
	}

	strategy := TrimEmptyOriginalPad
	if len(opts) > 0 && opts[0].EmptyOriginal != "" {
		strategy = opts[0].EmptyOriginal
	}
	if strategy != TrimEmptyOriginalPad && strategy != TrimEmptyOriginalKeep {
		return TrimResult{}, fmt.Errorf("%w: empty_original=%q", ErrInvalidTrimOptions, strategy)
	}

	start := 0
	for start < len(originalRunes) && start < len(revisedRunes) && originalRunes[start] == revisedRunes[start] {
		start++
	}

	originalEnd := len(originalRunes)
	revisedEnd := len(revisedRunes)
	for originalEnd > start && revisedEnd > start && originalRunes[originalEnd-1] == revisedRunes[revisedEnd-1] {
		originalEnd--
		revisedEnd--
	}

	revisedStart := start
	if start == originalEnd && strategy == TrimEmptyOriginalPad {
		if start > 0 {
			start--
			revisedStart--
		} else {
			originalEnd++
			revisedEnd++
		}
	}

	return TrimResult{
		Original: string(originalRunes[start:originalEnd]),
		Revised:  string(revisedRunes[revisedStart:revisedEnd]),
		Start:    start,
		End:      originalEnd,
	}, nil
}

func equalRunes(left, right []rune) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
