package textprocessor_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestMatchBatchDefaultsToLeftmostLongestAndDeduplicatesQueries(t *testing.T) {
	t.Parallel()

	got, err := textprocessor.MatchBatch("Go 项目和 Go", []string{"Go", "Go 项目", "Go"})
	if err != nil {
		t.Fatalf("MatchBatch() error = %v", err)
	}
	want := []textprocessor.BatchMatchResult{
		{Query: "Go 项目", Text: "Go 项目", Start: 0, End: 5},
		{Query: "Go", Text: "Go", Start: 7, End: 9},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MatchBatch() = %#v, want %#v", got, want)
	}
}

func TestMatchBatchCanReturnOverlappingMatches(t *testing.T) {
	t.Parallel()

	got, err := textprocessor.MatchBatch(
		"Go 项目",
		[]string{"Go", "Go 项目"},
		textprocessor.BatchMatchOptions{Overlapping: true},
	)
	if err != nil {
		t.Fatalf("MatchBatch() error = %v", err)
	}
	want := []textprocessor.BatchMatchResult{
		{Query: "Go 项目", Text: "Go 项目", Start: 0, End: 5},
		{Query: "Go", Text: "Go", Start: 0, End: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MatchBatch() = %#v, want %#v", got, want)
	}
}

func TestMatchBatchReturnsQueryAndActualASCIIInsensitiveText(t *testing.T) {
	t.Parallel()

	got, err := textprocessor.MatchBatch(
		"使用 GO 编写",
		[]string{"go"},
		textprocessor.BatchMatchOptions{ASCIIInsensitive: true},
	)
	if err != nil {
		t.Fatalf("MatchBatch() error = %v", err)
	}
	want := []textprocessor.BatchMatchResult{
		{Query: "go", Text: "GO", Start: 3, End: 5},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MatchBatch() = %#v, want %#v", got, want)
	}
}

func TestMatchBatchRejectsInvalidQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		queries []string
	}{
		{name: "empty query list"},
		{name: "empty query", queries: []string{"valid", ""}},
		{name: "invalid UTF-8", queries: []string{string([]byte{0xff})}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := textprocessor.MatchBatch("source", tt.queries)
			if !errors.Is(err, textprocessor.ErrInvalidBatchQueries) {
				t.Fatalf("MatchBatch() error = %v, want errors.Is(_, ErrInvalidBatchQueries)", err)
			}
		})
	}
}
