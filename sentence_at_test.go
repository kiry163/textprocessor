package textprocessor_test

import (
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestSentenceAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		text   string
		pos    int
		want   textprocessor.Span
		wantOK bool
	}{
		{
			name:   "first rune of first sentence",
			text:   "你好。Hello.",
			pos:    0,
			want:   textprocessor.Span{Text: "你好。", Start: 0, End: 3},
			wantOK: true,
		},
		{
			name:   "last rune of first sentence",
			text:   "你好。Hello.",
			pos:    2,
			want:   textprocessor.Span{Text: "你好。", Start: 0, End: 3},
			wantOK: true,
		},
		{
			name:   "boundary rune belongs to following sentence",
			text:   "你好。Hello.",
			pos:    3,
			want:   textprocessor.Span{Text: "Hello.", Start: 3, End: 9},
			wantOK: true,
		},
		{
			name:   "last rune of text",
			text:   "你好。Hello.",
			pos:    8,
			want:   textprocessor.Span{Text: "Hello.", Start: 3, End: 9},
			wantOK: true,
		},
		{
			name:   "interior rune of earlier repeated sentence",
			text:   "好。好。x",
			pos:    0,
			want:   textprocessor.Span{Text: "好。", Start: 0, End: 2},
			wantOK: true,
		},
		{
			name:   "later occurrence of repeated sentence",
			text:   "好。好。x",
			pos:    2,
			want:   textprocessor.Span{Text: "好。", Start: 2, End: 4},
			wantOK: true,
		},
		{
			name:   "unterminated trailing sentence",
			text:   "好。好。x",
			pos:    4,
			want:   textprocessor.Span{Text: "x", Start: 4, End: 5},
			wantOK: true,
		},
		{
			name:   "rune offsets for multibyte leading rune",
			text:   "😀Hi.好。",
			pos:    0,
			want:   textprocessor.Span{Text: "😀Hi.", Start: 0, End: 4},
			wantOK: true,
		},
		{
			name:   "rune offsets after multibyte leading rune",
			text:   "😀Hi.好。",
			pos:    5,
			want:   textprocessor.Span{Text: "好。", Start: 4, End: 6},
			wantOK: true,
		},
		{
			name:   "position at rune end of multibyte text",
			text:   "😀Hi.好。",
			pos:    6,
			wantOK: false,
		},
		{
			name:   "sentence on later line",
			text:   "第一句。\n\nSecond line.",
			pos:    10,
			want:   textprocessor.Span{Text: "Second line.", Start: 6, End: 18},
			wantOK: true,
		},
		{
			name:   "sentence before blank line",
			text:   "第一句。\n\nSecond line.",
			pos:    3,
			want:   textprocessor.Span{Text: "第一句。", Start: 0, End: 4},
			wantOK: true,
		},
		{
			name:   "newline before blank line",
			text:   "第一句。\n\nSecond line.",
			pos:    4,
			wantOK: false,
		},
		{
			name:   "newline after blank line",
			text:   "第一句。\n\nSecond line.",
			pos:    5,
			wantOK: false,
		},
		{
			name:   "leading indent",
			text:   "  缩进 你好。 尾随 ",
			pos:    1,
			wantOK: false,
		},
		{
			name:   "first sentence after indent",
			text:   "  缩进 你好。 尾随 ",
			pos:    2,
			want:   textprocessor.Span{Text: "缩进 你好。", Start: 2, End: 8},
			wantOK: true,
		},
		{
			name:   "whitespace between sentences",
			text:   "  缩进 你好。 尾随 ",
			pos:    8,
			wantOK: false,
		},
		{
			name:   "last sentence before trailing space",
			text:   "  缩进 你好。 尾随 ",
			pos:    9,
			want:   textprocessor.Span{Text: "尾随", Start: 9, End: 11},
			wantOK: true,
		},
		{
			name:   "trailing space",
			text:   "  缩进 你好。 尾随 ",
			pos:    11,
			wantOK: false,
		},
		{
			name:   "position at end of text",
			text:   "  缩进 你好。 尾随 ",
			pos:    12,
			wantOK: false,
		},
		{
			name:   "unterminated text is a sentence",
			text:   "abc",
			pos:    1,
			want:   textprocessor.Span{Text: "abc", Start: 0, End: 3},
			wantOK: true,
		},
		{
			name:   "negative position",
			text:   "你好。",
			pos:    -1,
			wantOK: false,
		},
		{
			name:   "invalid utf8 byte counts as one rune",
			text:   "\xff你好。",
			pos:    0,
			want:   textprocessor.Span{Text: "\ufffd你好。", Start: 0, End: 4},
			wantOK: true,
		},
		{
			name:   "rune after invalid utf8 byte",
			text:   "\xff你好。",
			pos:    3,
			want:   textprocessor.Span{Text: "\ufffd你好。", Start: 0, End: 4},
			wantOK: true,
		},
		{
			name:   "end of text with invalid utf8 byte",
			text:   "\xff你好。",
			pos:    4,
			wantOK: false,
		},
		{
			name:   "position past end",
			text:   "你好。",
			pos:    3,
			wantOK: false,
		},
		{
			name:   "empty text",
			text:   "",
			pos:    0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := textprocessor.SentenceAt(tt.text, tt.pos)
			if ok != tt.wantOK {
				t.Fatalf("SentenceAt(%q, %d) ok = %v, want %v", tt.text, tt.pos, ok, tt.wantOK)
			}
			if !tt.wantOK {
				if got != (textprocessor.Span{}) {
					t.Fatalf("SentenceAt(%q, %d) = %#v, want zero Span", tt.text, tt.pos, got)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("SentenceAt(%q, %d) = %#v, want %#v", tt.text, tt.pos, got, tt.want)
			}

			runes := []rune(tt.text)
			if tt.pos < got.Start || tt.pos >= got.End {
				t.Fatalf("SentenceAt(%q, %d) = %#v does not contain pos", tt.text, tt.pos, got)
			}
			if got.Text != string(runes[got.Start:got.End]) {
				t.Fatalf("SentenceAt(%q, %d) Text = %q, want %q", tt.text, tt.pos, got.Text, string(runes[got.Start:got.End]))
			}
		})
	}
}
