package textprocessor_test

import (
	"errors"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestTrimDifference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		revised  string
		opts     []textprocessor.TrimOptions
		want     textprocessor.TrimResult
	}{
		{
			name:     "replacement",
			original: "abc",
			revised:  "axc",
			want:     textprocessor.TrimResult{Original: "b", Revised: "x", Start: 1, End: 2},
		},
		{
			name:     "deletion",
			original: "abc",
			revised:  "ac",
			want:     textprocessor.TrimResult{Original: "b", Revised: "", Start: 1, End: 2},
		},
		{
			name:     "delete entire original",
			original: "abc",
			revised:  "",
			want:     textprocessor.TrimResult{Original: "abc", Revised: "", Start: 0, End: 3},
		},
		{
			name:     "complete replacement",
			original: "甲乙",
			revised:  "XY",
			want:     textprocessor.TrimResult{Original: "甲乙", Revised: "XY", Start: 0, End: 2},
		},
		{
			name:     "middle insertion explicitly pads left",
			original: "abc",
			revised:  "abXc",
			opts: []textprocessor.TrimOptions{{
				EmptyOriginal: textprocessor.TrimEmptyOriginalPad,
			}},
			want: textprocessor.TrimResult{Original: "b", Revised: "bX", Start: 1, End: 2},
		},
		{
			name:     "start insertion falls back to right",
			original: "abc",
			revised:  "Xabc",
			want:     textprocessor.TrimResult{Original: "a", Revised: "Xa", Start: 0, End: 1},
		},
		{
			name:     "zero options use default padding",
			original: "abc",
			revised:  "abcX",
			opts:     []textprocessor.TrimOptions{{}},
			want:     textprocessor.TrimResult{Original: "c", Revised: "cX", Start: 2, End: 3},
		},
		{
			name:     "keep empty insertion range",
			original: "abc",
			revised:  "abXc",
			opts: []textprocessor.TrimOptions{{
				EmptyOriginal: textprocessor.TrimEmptyOriginalKeep,
			}},
			want: textprocessor.TrimResult{Original: "", Revised: "X", Start: 2, End: 2},
		},
		{
			name:     "emoji uses rune offsets",
			original: "你🙂好",
			revised:  "你🙂们好",
			want:     textprocessor.TrimResult{Original: "🙂", Revised: "🙂们", Start: 1, End: 2},
		},
		{
			name:     "punctuation and whitespace are compared exactly",
			original: "a， b",
			revised:  "a！ b",
			want:     textprocessor.TrimResult{Original: "，", Revised: "！", Start: 1, End: 2},
		},
		{
			name:     "one rune original supports start insertion",
			original: "a",
			revised:  "Xa",
			want:     textprocessor.TrimResult{Original: "a", Revised: "Xa", Start: 0, End: 1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := textprocessor.TrimDifference(tt.original, tt.revised, tt.opts...)
			if err != nil {
				t.Fatalf("TrimDifference() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("TrimDifference() = %#v, want %#v", got, tt.want)
			}

			originalRunes := []rune(tt.original)
			if got.Start < 0 || got.Start > got.End || got.End > len(originalRunes) {
				t.Fatalf("TrimDifference() range = [%d,%d), original rune length = %d", got.Start, got.End, len(originalRunes))
			}
			if fragment := string(originalRunes[got.Start:got.End]); fragment != got.Original {
				t.Fatalf("original range contains %q, result Original = %q", fragment, got.Original)
			}
			rebuilt := string(originalRunes[:got.Start]) + got.Revised + string(originalRunes[got.End:])
			if rebuilt != tt.revised {
				t.Fatalf("replacing result range rebuilt %q, want %q", rebuilt, tt.revised)
			}
		})
	}
}

func TestTrimDifferenceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		revised  string
		opts     []textprocessor.TrimOptions
		want     error
	}{
		{
			name:     "empty original",
			original: "",
			revised:  "X",
			want:     textprocessor.ErrOriginalEmpty,
		},
		{
			name:     "both inputs empty",
			original: "",
			revised:  "",
			want:     textprocessor.ErrOriginalEmpty,
		},
		{
			name:     "no change",
			original: "abc",
			revised:  "abc",
			want:     textprocessor.ErrNoChange,
		},
		{
			name:     "invalid strategy",
			original: "abc",
			revised:  "axc",
			opts: []textprocessor.TrimOptions{{
				EmptyOriginal: textprocessor.TrimEmptyOriginalStrategy("invalid"),
			}},
			want: textprocessor.ErrInvalidTrimOptions,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := textprocessor.TrimDifference(tt.original, tt.revised, tt.opts...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("TrimDifference() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}
