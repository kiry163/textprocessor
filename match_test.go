package textspan_test

import (
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestMatchExactReturnsAllExactMatchesWithRuneOffsets(t *testing.T) {
	source := "前缀 ABC 后缀 ABC"

	got := textspan.Match(source, "ABC", textspan.MatchOptions{Mode: textspan.MatchExact})
	want := []textspan.MatchResult{
		{Text: "ABC", Start: 3, End: 6},
		{Text: "ABC", Start: 10, End: 13},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchNormalizedIgnoresPunctuationAndWhitespace(t *testing.T) {
	source := "他说：Hello, world！然后 Hello world。"

	got := textspan.Match(source, "Hello world", textspan.MatchOptions{Mode: textspan.MatchNormalized})
	want := []textspan.MatchResult{
		{Text: "Hello, world", Start: 3, End: 15},
		{Text: "Hello world", Start: 19, End: 30},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchSmartPrefersExactMatches(t *testing.T) {
	source := "Hello, world. Hello world."

	got := textspan.Match(source, "Hello world", textspan.MatchOptions{Mode: textspan.MatchSmart})
	want := []textspan.MatchResult{
		{Text: "Hello world", Start: 14, End: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchSmartFallsBackToNormalizedMatches(t *testing.T) {
	source := "他说：Hello, world！"

	got := textspan.Match(source, "Hello world")
	want := []textspan.MatchResult{
		{Text: "Hello, world", Start: 3, End: 15},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchNormalizedRecoversIgnoredQueryEdgeFromSource(t *testing.T) {
	source := "为顺利实现支护安全风险的全面无死角封盖，除要充分关注支护结构本体外，亦需要有意识将重点延伸至四周环境，真正做到统筹兼顾。"

	got := textspan.Match(source, "封盖。", textspan.MatchOptions{Mode: textspan.MatchNormalized})
	want := []textspan.MatchResult{
		{Text: "封盖，", Start: 17, End: 20},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchBoundaryNoneDoesNotRecoverIgnoredQueryEdge(t *testing.T) {
	source := "为顺利实现支护安全风险的全面无死角封盖，除要充分关注支护结构本体外，亦需要有意识将重点延伸至四周环境，真正做到统筹兼顾。"

	got := textspan.Match(source, "封盖。", textspan.MatchOptions{
		Mode:     textspan.MatchNormalized,
		Boundary: textspan.MatchBoundaryNone,
	})
	want := []textspan.MatchResult{
		{Text: "封盖", Start: 17, End: 19},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %#v, want %#v", got, want)
	}
}

func TestMatchReturnsEmptySliceWhenFragmentIsAbsent(t *testing.T) {
	got := textspan.Match("alpha beta", "gamma")

	if len(got) != 0 {
		t.Fatalf("Match() = %#v, want empty slice", got)
	}
}
