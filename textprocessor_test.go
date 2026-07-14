package textprocessor_test

import (
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestSentencesSegmentsChineseEnglishMixedText(t *testing.T) {
	text := "这是第一句。This is sentence two. 我见过 Mr. Smith。他说 hello world."

	got := textprocessor.Sentences(text)
	want := []string{
		"这是第一句。",
		"This is sentence two.",
		"我见过 Mr. Smith。",
		"他说 hello world.",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sentences() = %#v, want %#v", got, want)
	}
}

func TestSentencesKeepsEnglishPunctuationInsideChineseBookTitle(t *testing.T) {
	text := "我们明天看《Hello! World》好吗？Yes. Let's go."

	got := textprocessor.Sentences(text)
	want := []string{
		"我们明天看《Hello! World》好吗？",
		"Yes.",
		"Let's go.",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sentences() = %#v, want %#v", got, want)
	}
}
