package textprocessor_test

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestRepeatedSubstringsReturnsSupermaximalRepeats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		minChars int
		want     []string
	}{
		{name: "no repeats", text: "abcdef", minChars: 2},
		{name: "empty text", text: "", minChars: 1},
		{name: "single rune", text: "a", minChars: 1},
		{name: "all same runes keep longest", text: "aaaa", minChars: 2, want: []string{"aaa"}},
		{name: "all same runes min one", text: "aaaa", minChars: 1, want: []string{"aaa"}},
		{name: "whole text occurs once", text: "aaaa", minChars: 4},
		{name: "overlapping repeat", text: "abab", minChars: 1, want: []string{"ab"}},
		{name: "repeated block keeps longest", text: "abcabcXabcabc", minChars: 3, want: []string{"abcabc"}},
		{name: "repeated block at exact min", text: "abcabcXabcabc", minChars: 6, want: []string{"abcabc"}},
		{name: "repeated block below min", text: "abcabcXabcabc", minChars: 7},
		{name: "CJK counts runes", text: "你好你好", minChars: 2, want: []string{"你好"}},
		{name: "CJK below min", text: "你好你好", minChars: 3},
		{name: "emoji is one rune", text: "😀😀", minChars: 1, want: []string{"😀"}},
		{
			name:     "longer result first",
			text:     "abcdabcdXabcdabcdYabcabcZabcabc",
			minChars: 3,
			want:     []string{"abcdabcd", "abcabc"},
		},
		{
			name:     "equal length ordered by position",
			text:     "abcabcXabcabcYdefdefZdefdef",
			minChars: 3,
			want:     []string{"abcabc", "defdef"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := textprocessor.RepeatedSubstrings(tt.text, tt.minChars)
			if err != nil {
				t.Fatalf("RepeatedSubstrings() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("RepeatedSubstrings(%q, %d) = %#v, want %#v", tt.text, tt.minChars, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("RepeatedSubstrings(%q, %d) = %#v, want %#v", tt.text, tt.minChars, got, tt.want)
				}
			}
		})
	}
}

func TestRepeatedSubstringsFindsDuplicatedBlock(t *testing.T) {
	t.Parallel()

	half := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 100) + "0123456789"
	got, err := textprocessor.RepeatedSubstrings(half+half, 1)
	if err != nil {
		t.Fatalf("RepeatedSubstrings() error = %v", err)
	}
	if len(got) != 1 || got[0] != half {
		t.Fatalf("RepeatedSubstrings() = %d results, want 1 result of %d runes", len(got), len([]rune(half)))
	}
}

func TestRepeatedSubstringsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		minChars int
	}{
		{name: "invalid utf8", text: string([]byte{0xff}), minChars: 1},
		{name: "zero min chars", text: "aa", minChars: 0},
		{name: "negative min chars", text: "aa", minChars: -1},
		{
			name:     "over rune limit",
			text:     strings.Repeat("a", textprocessor.MaxRepeatedSubstringRunes+1),
			minChars: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := textprocessor.RepeatedSubstrings(tt.text, tt.minChars)
			if !errors.Is(err, textprocessor.ErrInvalidRepeatInput) {
				t.Fatalf("RepeatedSubstrings() error = %v, want %v", err, textprocessor.ErrInvalidRepeatInput)
			}
			if got != nil {
				t.Fatalf("RepeatedSubstrings() = %#v, want nil", got)
			}
		})
	}
}

func TestRepeatedSubstringsMatchesBruteForce(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(20261021, 1))
	alphabet := []rune("abc")

	for i := 0; i < 300; i++ {
		n := rng.IntN(13)
		runes := make([]rune, n)
		for j := range runes {
			runes[j] = alphabet[rng.IntN(len(alphabet))]
		}
		text := string(runes)
		minChars := 1 + rng.IntN(4)

		got, err := textprocessor.RepeatedSubstrings(text, minChars)
		if err != nil {
			t.Fatalf("RepeatedSubstrings(%q, %d) error = %v", text, minChars, err)
		}
		want := repeatBruteForce(runes, minChars)
		if !repeatSameStringSet(got, want) {
			t.Fatalf("RepeatedSubstrings(%q, %d) = %#v, want set %#v", text, minChars, got, want)
		}
	}
}

func repeatBruteForce(runes []rune, minChars int) []string {
	n := len(runes)
	occurrences := make(map[string][]int)
	lengths := make(map[string]int)
	for start := 0; start < n; start++ {
		for end := start + minChars; end <= n; end++ {
			text := string(runes[start:end])
			occurrences[text] = append(occurrences[text], start)
			lengths[text] = end - start
		}
	}

	maximal := make(map[string]bool)
	for text, starts := range occurrences {
		if len(starts) < 2 {
			continue
		}
		if repeatLeftDiverse(runes, starts) && repeatRightDiverse(runes, starts, lengths[text]) {
			maximal[text] = true
		}
	}

	var result []string
	for text := range maximal {
		contained := false
		for longer := range maximal {
			if lengths[longer] > lengths[text] && strings.Contains(longer, text) {
				contained = true
				break
			}
		}
		if !contained {
			result = append(result, text)
		}
	}
	return result
}

func repeatLeftDiverse(runes []rune, starts []int) bool {
	first := repeatLeftChar(runes, starts[0])
	for _, start := range starts[1:] {
		if repeatLeftChar(runes, start) != first {
			return true
		}
	}
	return false
}

func repeatLeftChar(runes []rune, start int) int {
	if start == 0 {
		return -1
	}
	return int(runes[start-1])
}

func repeatRightDiverse(runes []rune, starts []int, length int) bool {
	first := repeatRightChar(runes, starts[0], length)
	for _, start := range starts[1:] {
		if repeatRightChar(runes, start, length) != first {
			return true
		}
	}
	return false
}

func repeatRightChar(runes []rune, start, length int) int {
	if start+length == len(runes) {
		return -1
	}
	return int(runes[start+length])
}

func repeatSameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(want))
	for _, text := range want {
		counts[text]++
	}
	for _, text := range got {
		if counts[text] == 0 {
			return false
		}
		counts[text]--
	}
	return true
}

func BenchmarkRepeatedSubstrings(b *testing.B) {
	makeRandom := func(n int, seed int64) string {
		rng := rand.New(rand.NewPCG(uint64(seed), 1))
		alphabet := []rune("abcdefghijklmnopqrstuvwxyz")
		runes := make([]rune, n)
		for i := range runes {
			runes[i] = alphabet[rng.IntN(len(alphabet))]
		}
		return string(runes)
	}
	randomHalf := makeRandom(textprocessor.MaxRepeatedSubstringRunes/2, 1)

	cases := []struct {
		name     string
		text     string
		minChars int
	}{
		{name: "random 1m", text: makeRandom(textprocessor.MaxRepeatedSubstringRunes, 2), minChars: 20},
		{name: "duplicated half", text: randomHalf + randomHalf, minChars: 20},
		{name: "same rune 1m", text: strings.Repeat("a", textprocessor.MaxRepeatedSubstringRunes), minChars: 20},
	}

	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := textprocessor.RepeatedSubstrings(bc.text, bc.minChars); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
