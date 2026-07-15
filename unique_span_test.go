package textprocessor_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestBuildUniqueSpansKeepsAlreadyUniqueText(t *testing.T) {
	t.Parallel()

	got, err := textprocessor.BuildUniqueSpans("前缀目标后缀", []textprocessor.Span{
		{Text: "目标", Start: 2, End: 4},
	})
	if err != nil {
		t.Fatalf("BuildUniqueSpans() error = %v", err)
	}
	want := []textprocessor.UniqueSpan{
		{Text: "目标", Start: 2, End: 4, Unique: "目标"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildUniqueSpans() = %#v, want %#v", got, want)
	}
}

func TestBuildUniqueSpansAlternatesLeftAndRightInBatches(t *testing.T) {
	t.Parallel()

	source := "甲A目标X乙A目标Y"
	got, err := textprocessor.BuildUniqueSpans(source, []textprocessor.Span{
		{Text: "目标", Start: 2, End: 4},
		{Text: "目标", Start: 7, End: 9},
	})
	if err != nil {
		t.Fatalf("BuildUniqueSpans() error = %v", err)
	}
	want := []textprocessor.UniqueSpan{
		{Text: "目标", Start: 2, End: 4, Unique: "A目标X"},
		{Text: "目标", Start: 7, End: 9, Unique: "A目标Y"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildUniqueSpans() = %#v, want %#v", got, want)
	}
}

func TestBuildUniqueSpansFallsBackAtEachSourceBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		span   textprocessor.Span
		want   string
	}{
		{
			name:   "left boundary expands right",
			source: "目标X目标Y",
			span:   textprocessor.Span{Text: "目标", Start: 0, End: 2},
			want:   "目标X",
		},
		{
			name:   "right boundary expands left",
			source: "A目标A目标",
			span:   textprocessor.Span{Text: "目标", Start: 4, End: 6},
			want:   "标A目标",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := textprocessor.BuildUniqueSpans(tt.source, []textprocessor.Span{tt.span})
			if err != nil {
				t.Fatalf("BuildUniqueSpans() error = %v", err)
			}
			if len(got) != 1 || got[0].Unique != tt.want {
				t.Fatalf("BuildUniqueSpans() = %#v, want Unique %q", got, tt.want)
			}
		})
	}
}

func TestBuildUniqueSpansCountsOverlappingOccurrences(t *testing.T) {
	t.Parallel()

	got, err := textprocessor.BuildUniqueSpans("aaaa", []textprocessor.Span{
		{Text: "aa", Start: 0, End: 2},
	})
	if err != nil {
		t.Fatalf("BuildUniqueSpans() error = %v", err)
	}
	want := []textprocessor.UniqueSpan{
		{Text: "aa", Start: 0, End: 2, Unique: "aaaa"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildUniqueSpans() = %#v, want %#v", got, want)
	}
}

func TestBuildUniqueSpansRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		span textprocessor.Span
	}{
		{name: "negative start", span: textprocessor.Span{Text: "a", Start: -1, End: 1}},
		{name: "empty range", span: textprocessor.Span{Text: "", Start: 1, End: 1}},
		{name: "end past source", span: textprocessor.Span{Text: "a", Start: 0, End: 4}},
		{name: "text mismatch", span: textprocessor.Span{Text: "x", Start: 0, End: 1}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := textprocessor.BuildUniqueSpans("abc", []textprocessor.Span{tt.span})
			if !errors.Is(err, textprocessor.ErrInvalidSpan) {
				t.Fatalf("BuildUniqueSpans() error = %v, want errors.Is(_, ErrInvalidSpan)", err)
			}
		})
	}
}

func TestBuildUniqueSpansRejectsInvalidUTF8Source(t *testing.T) {
	t.Parallel()

	_, err := textprocessor.BuildUniqueSpans(string([]byte{0xff}), nil)
	if !errors.Is(err, textprocessor.ErrInvalidSpan) {
		t.Fatalf("BuildUniqueSpans() error = %v, want errors.Is(_, ErrInvalidSpan)", err)
	}
}
