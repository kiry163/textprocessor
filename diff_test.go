package textprocessor_test

import (
	"errors"
	"testing"
	"unicode/utf8"

	"github.com/kiry163/textprocessor"
)

func TestDiffReconstructsBothTexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		revised  string
	}{
		{name: "multiple edits", original: "我喜欢苹果和香蕉", revised: "我喜欢红苹果和葡萄"},
		{name: "emoji", original: "A🙂中文B", revised: "A🙂新中文!B"},
		{name: "insert only", original: "abc", revised: "abXYZc"},
		{name: "delete only", original: "abXYZc", revised: "abc"},
		{name: "complete replacement", original: "旧文本", revised: "new text"},
		{name: "empty original", original: "", revised: "新增"},
		{name: "empty revised", original: "删除", revised: ""},
		{name: "both empty", original: "", revised: ""},
		{name: "identical", original: "相同🙂", revised: "相同🙂"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parts, err := textprocessor.Diff(tt.original, tt.revised)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}
			assertDiffContract(t, tt.original, tt.revised, parts)
		})
	}
}

func TestDiffRangesUseRuneOffsets(t *testing.T) {
	t.Parallel()

	parts, err := textprocessor.Diff("你🙂好", "你🙂们好", textprocessor.DiffOptions{
		Cleanup: textprocessor.DiffCleanupNone,
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	want := []textprocessor.DiffPart{
		{Operation: textprocessor.DiffEqual, Text: "你🙂", OriginalStart: 0, OriginalEnd: 2, RevisedStart: 0, RevisedEnd: 2},
		{Operation: textprocessor.DiffInsert, Text: "们", OriginalStart: 2, OriginalEnd: 2, RevisedStart: 2, RevisedEnd: 3},
		{Operation: textprocessor.DiffEqual, Text: "好", OriginalStart: 2, OriginalEnd: 3, RevisedStart: 3, RevisedEnd: 4},
	}
	if !equalDiffParts(parts, want) {
		t.Fatalf("Diff() = %#v, want %#v", parts, want)
	}
}

func TestDiffOptions(t *testing.T) {
	t.Parallel()

	for _, options := range []textprocessor.DiffOptions{
		{Cleanup: textprocessor.DiffCleanupMode("invalid")},
		{Timeout: -1},
	} {
		_, err := textprocessor.Diff("a", "b", options)
		if !errors.Is(err, textprocessor.ErrInvalidDiffOptions) {
			t.Fatalf("Diff() error = %v, want errors.Is(_, ErrInvalidDiffOptions)", err)
		}
	}
}

func TestDiffRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	invalid := string([]byte{0xff})
	for _, inputs := range [][2]string{{invalid, "valid"}, {"valid", invalid}} {
		_, err := textprocessor.Diff(inputs[0], inputs[1])
		if !errors.Is(err, textprocessor.ErrInvalidDiffInput) {
			t.Fatalf("Diff() error = %v, want errors.Is(_, ErrInvalidDiffInput)", err)
		}
	}
}

func assertDiffContract(t *testing.T, original, revised string, parts []textprocessor.DiffPart) {
	t.Helper()

	originalRunes := []rune(original)
	revisedRunes := []rune(revised)
	originalPos := 0
	revisedPos := 0
	originalText := ""
	revisedText := ""
	for i, part := range parts {
		if part.Text == "" {
			t.Fatalf("part %d has empty text", i)
		}
		if part.OriginalStart != originalPos || part.RevisedStart != revisedPos {
			t.Fatalf("part %d starts at original/revised %d/%d, want %d/%d", i, part.OriginalStart, part.RevisedStart, originalPos, revisedPos)
		}
		n := utf8.RuneCountInString(part.Text)
		switch part.Operation {
		case textprocessor.DiffEqual:
			if part.OriginalEnd-part.OriginalStart != n || part.RevisedEnd-part.RevisedStart != n {
				t.Fatalf("equal part %d ranges do not match rune length %d", i, n)
			}
			originalText += part.Text
			revisedText += part.Text
		case textprocessor.DiffDelete:
			if part.OriginalEnd-part.OriginalStart != n || part.RevisedEnd != part.RevisedStart {
				t.Fatalf("delete part %d has invalid ranges", i)
			}
			originalText += part.Text
		case textprocessor.DiffInsert:
			if part.OriginalEnd != part.OriginalStart || part.RevisedEnd-part.RevisedStart != n {
				t.Fatalf("insert part %d has invalid ranges", i)
			}
			revisedText += part.Text
		default:
			t.Fatalf("part %d has unknown operation %q", i, part.Operation)
		}
		originalPos = part.OriginalEnd
		revisedPos = part.RevisedEnd
	}
	if originalPos != len(originalRunes) || revisedPos != len(revisedRunes) {
		t.Fatalf("final positions = %d/%d, want %d/%d", originalPos, revisedPos, len(originalRunes), len(revisedRunes))
	}
	if originalText != original || revisedText != revised {
		t.Fatalf("reconstructed original/revised = %q/%q, want %q/%q", originalText, revisedText, original, revised)
	}
}

func equalDiffParts(left, right []textprocessor.DiffPart) bool {
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
