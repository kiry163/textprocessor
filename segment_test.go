package textspan_test

import (
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestSegmentReturnsRuneOffsetsForChineseEnglishMixedText(t *testing.T) {
	text := "你好。Hello."

	got := textspan.Segment(text, textspan.SegmentOptions{MinChineseChars: 0})
	want := []textspan.Span{
		{Text: "你好。", Start: 0, End: 3},
		{Text: "Hello.", Start: 3, End: 9},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segment() = %#v, want %#v", got, want)
	}
}

func TestSegmentSkipsBlankLinesAndPreservesLineOffsets(t *testing.T) {
	text := "第一句。\n\nSecond line."

	got := textspan.Segment(text, textspan.SegmentOptions{MinChineseChars: 0})
	want := []textspan.Span{
		{Text: "第一句。", Start: 0, End: 4},
		{Text: "Second line.", Start: 6, End: 18},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segment() = %#v, want %#v", got, want)
	}
}

func TestSegmentMergesShortChineseSpansWithFollowingSpan(t *testing.T) {
	text := "短句。第二个句子很长。Final."

	got := textspan.Segment(text, textspan.SegmentOptions{MinChineseChars: 5})
	want := []textspan.Span{
		{Text: "短句。第二个句子很长。", Start: 0, End: 11},
		{Text: "Final.", Start: 11, End: 17},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segment() = %#v, want %#v", got, want)
	}
}
