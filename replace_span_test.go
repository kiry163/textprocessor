package textprocessor_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

// assertReplaceSpansInvariant checks the core contract of ReplaceSpans: every
// returned span is an ascending, non-overlapping range whose content in the
// returned text equals its Text field.
func assertReplaceSpansInvariant(t *testing.T, text string, spans []textprocessor.ReplaceSpan[string]) {
	t.Helper()

	runes := []rune(text)
	prevEnd := 0
	for i, span := range spans {
		if span.Start < prevEnd || span.End < span.Start || span.End > len(runes) {
			t.Fatalf("span %d = [%d,%d) is out of order or out of range for %d runes", i, span.Start, span.End, len(runes))
		}
		if got := string(runes[span.Start:span.End]); got != span.Text {
			t.Fatalf("text[%d:%d] = %q, span %d Text = %q", span.Start, span.End, got, i, span.Text)
		}
		prevEnd = span.End
	}
}

func TestReplaceSpansRewritesTextAndOffsets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		parts     []textprocessor.ReplaceSpan[string]
		wantText  string
		wantSpans []textprocessor.ReplaceSpan[string]
	}{
		{
			name:   "replaces multiple spans and sorts them by start",
			source: "你好。Hello.",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "s2", Start: 3, End: 9, Text: "Hello!"},
				{ID: "s1", Start: 0, End: 3, Text: "你好！"},
			},
			wantText: "你好！Hello!",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "s1", Start: 0, End: 3, Text: "你好！"},
				{ID: "s2", Start: 3, End: 9, Text: "Hello!"},
			},
		},
		{
			name:   "rewrites offsets after a longer replacement",
			source: "他说：Hello, world！",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "t", Start: 3, End: 15, Text: "你好，世界"},
			},
			wantText: "他说：你好，世界！",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "t", Start: 3, End: 8, Text: "你好，世界"},
			},
		},
		{
			name:   "rewrites offsets after a shorter replacement",
			source: "你好。Hello.",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "s", Start: 3, End: 9, Text: "Hi"},
			},
			wantText: "你好。Hi",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "s", Start: 3, End: 5, Text: "Hi"},
			},
		},
		{
			name:   "deletes the range when the replacement is empty",
			source: "abc",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "d", Start: 1, End: 2, Text: ""},
			},
			wantText: "ac",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "d", Start: 1, End: 1, Text: ""},
			},
		},
		{
			name:   "inserts when the range is empty",
			source: "abc",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 2, End: 2, Text: "XY"},
			},
			wantText: "abXYc",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 2, End: 4, Text: "XY"},
			},
		},
		{
			name:   "keeps gaps and tail verbatim",
			source: "abcdefgh",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 0, End: 2, Text: "X"},
				{ID: "c", Start: 8, End: 8, Text: "ZZ"},
				{ID: "b", Start: 4, End: 6, Text: ""},
			},
			wantText: "XcdghZZ",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 0, End: 1, Text: "X"},
				{ID: "b", Start: 3, End: 3, Text: ""},
				{ID: "c", Start: 5, End: 7, Text: "ZZ"},
			},
		},
		{
			name:   "treats adjacent spans as non-overlapping",
			source: "abcdef",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "b", Start: 3, End: 5, Text: "Y"},
				{ID: "a", Start: 0, End: 3, Text: "X"},
			},
			wantText: "XYf",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 0, End: 1, Text: "X"},
				{ID: "b", Start: 1, End: 2, Text: "Y"},
			},
		},
		{
			name:   "accepts a zero length span at a replaced boundary",
			source: "abcdef",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 3, End: 3, Text: "I"},
				{ID: "r", Start: 0, End: 3, Text: "R"},
			},
			wantText: "RIdef",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "r", Start: 0, End: 1, Text: "R"},
				{ID: "i", Start: 1, End: 2, Text: "I"},
			},
		},
		{
			name:   "sorts a zero length span before a span at the same start",
			source: "abcdef",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "r", Start: 2, End: 4, Text: "R"},
				{ID: "i", Start: 2, End: 2, Text: "I"},
			},
			wantText: "abIRef",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 2, End: 3, Text: "I"},
				{ID: "r", Start: 3, End: 4, Text: "R"},
			},
		},
		{
			name:   "keeps caller order for equal zero length spans",
			source: "abc",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "A", Start: 1, End: 1, Text: "A"},
				{ID: "B", Start: 1, End: 1, Text: "B"},
			},
			wantText: "aABbc",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "A", Start: 1, End: 2, Text: "A"},
				{ID: "B", Start: 2, End: 3, Text: "B"},
			},
		},
		{
			name:   "uses rune offsets for multi byte text",
			source: "你🙂好",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "e", Start: 1, End: 2, Text: "X"},
			},
			wantText: "你X好",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "e", Start: 1, End: 2, Text: "X"},
			},
		},
		{
			name:   "inserts into an empty source",
			source: "",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 0, End: 0, Text: "新"},
			},
			wantText: "新",
			wantSpans: []textprocessor.ReplaceSpan[string]{
				{ID: "i", Start: 0, End: 1, Text: "新"},
			},
		},
		{
			name:      "returns the source unchanged without spans",
			source:    "原样文本",
			parts:     nil,
			wantText:  "原样文本",
			wantSpans: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotText, gotSpans, err := textprocessor.ReplaceSpans(tt.source, tt.parts)
			if err != nil {
				t.Fatalf("ReplaceSpans() error = %v", err)
			}
			if gotText != tt.wantText {
				t.Fatalf("ReplaceSpans() text = %q, want %q", gotText, tt.wantText)
			}
			if !reflect.DeepEqual(gotSpans, tt.wantSpans) {
				t.Fatalf("ReplaceSpans() spans = %#v, want %#v", gotSpans, tt.wantSpans)
			}
			assertReplaceSpansInvariant(t, gotText, gotSpans)
		})
	}
}

func TestReplaceSpansUpdatesInputSpansInPlace(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.ReplaceSpan[string]{
		{ID: "a", Start: 0, End: 3, Text: "XY"},
		{ID: "b", Start: 3, End: 6, Text: "Z"},
	}
	text, spans, err := textprocessor.ReplaceSpans("abc123", parts)
	if err != nil {
		t.Fatalf("ReplaceSpans() error = %v", err)
	}
	if text != "XYZ" {
		t.Fatalf("ReplaceSpans() text = %q, want %q", text, "XYZ")
	}
	want := []textprocessor.ReplaceSpan[string]{
		{ID: "a", Start: 0, End: 2, Text: "XY"},
		{ID: "b", Start: 2, End: 3, Text: "Z"},
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("ReplaceSpans() left caller spans = %#v, want %#v", parts, want)
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("ReplaceSpans() returned spans = %#v, want %#v", spans, want)
	}
}

func TestReplaceSpansPreservesCallerSuppliedID(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.ReplaceSpan[int64]{
		{ID: 42, Start: 0, End: 3, Text: "X"},
		{ID: 7, Start: 3, End: 6, Text: "YY"},
	}
	text, spans, err := textprocessor.ReplaceSpans("abcdef", parts)
	if err != nil {
		t.Fatalf("ReplaceSpans() error = %v", err)
	}
	if text != "XYY" {
		t.Fatalf("ReplaceSpans() text = %q, want %q", text, "XYY")
	}
	want := []textprocessor.ReplaceSpan[int64]{
		{ID: 42, Start: 0, End: 1, Text: "X"},
		{ID: 7, Start: 1, End: 3, Text: "YY"},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("ReplaceSpans() spans = %#v, want %#v", spans, want)
	}
}

func TestReplaceSpansRejectsInvalidSpans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		parts  []textprocessor.ReplaceSpan[string]
	}{
		{
			name:   "negative start",
			source: "abc",
			parts:  []textprocessor.ReplaceSpan[string]{{ID: "a", Start: -1, End: 1, Text: "X"}},
		},
		{
			name:   "start after end",
			source: "abc",
			parts:  []textprocessor.ReplaceSpan[string]{{ID: "a", Start: 2, End: 1, Text: "X"}},
		},
		{
			name:   "end beyond rune length",
			source: "你好",
			parts:  []textprocessor.ReplaceSpan[string]{{ID: "a", Start: 0, End: 3, Text: "X"}},
		},
		{
			name:   "invalid utf-8 source",
			source: "\xff\xfe",
			parts:  []textprocessor.ReplaceSpan[string]{{ID: "a", Start: 0, End: 1, Text: "X"}},
		},
		{
			name:   "invalid utf-8 source without spans",
			source: "\xff",
			parts:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := append([]textprocessor.ReplaceSpan[string](nil), tt.parts...)
			text, _, err := textprocessor.ReplaceSpans(tt.source, tt.parts)
			if !errors.Is(err, textprocessor.ErrInvalidReplaceSpan) {
				t.Fatalf("ReplaceSpans() error = %v, want errors.Is(_, %v)", err, textprocessor.ErrInvalidReplaceSpan)
			}
			if text != "" {
				t.Fatalf("ReplaceSpans() text = %q, want empty text on error", text)
			}
			if !reflect.DeepEqual(tt.parts, before) {
				t.Fatalf("ReplaceSpans() modified caller spans on error: %#v, want %#v", tt.parts, before)
			}
		})
	}
}

func TestReplaceSpansRejectsOverlappingSpans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []textprocessor.ReplaceSpan[string]
	}{
		{
			name: "partial overlap",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 0, End: 3, Text: "X"},
				{ID: "b", Start: 2, End: 5, Text: "Y"},
			},
		},
		{
			name: "contained span",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 0, End: 4, Text: "X"},
				{ID: "b", Start: 1, End: 2, Text: "Y"},
			},
		},
		{
			name: "identical spans",
			parts: []textprocessor.ReplaceSpan[string]{
				{ID: "a", Start: 1, End: 3, Text: "X"},
				{ID: "b", Start: 1, End: 3, Text: "Y"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := append([]textprocessor.ReplaceSpan[string](nil), tt.parts...)
			text, _, err := textprocessor.ReplaceSpans("abcdef", tt.parts)
			if !errors.Is(err, textprocessor.ErrInvalidReplaceSpan) {
				t.Fatalf("ReplaceSpans() error = %v, want errors.Is(_, %v)", err, textprocessor.ErrInvalidReplaceSpan)
			}
			if text != "" {
				t.Fatalf("ReplaceSpans() text = %q, want empty text on error", text)
			}
			if !reflect.DeepEqual(tt.parts, before) {
				t.Fatalf("ReplaceSpans() modified caller spans on error: %#v, want %#v", tt.parts, before)
			}
		})
	}
}
