package textprocessor

import (
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"
)

// MaxRepeatedSubstringRunes is the maximum rune length accepted by
// RepeatedSubstrings.
const MaxRepeatedSubstringRunes = 1_000_000

// ErrInvalidRepeatInput indicates invalid input to RepeatedSubstrings.
var ErrInvalidRepeatInput = errors.New("invalid repeated substring input")

// repeatPairwiseLimit keeps the common small occurrence set free of sorting
// allocations; larger sets are sorted before the distinctness check.
const repeatPairwiseLimit = 32

// RepeatedSubstrings returns the supermaximal repeated substrings of text
// whose rune length is at least minChars.
//
// A returned substring occurs at least twice, is left- and right-maximal, and
// is not contained in any longer repeated substring. No returned substring is
// contained in another returned substring. Overlapping occurrences count as
// repeats.
//
// Results are ordered by rune length descending, then by first occurrence.
// Lengths are measured in runes. Returned strings share text's backing memory,
// so retaining them keeps text alive.
//
// RepeatedSubstrings rejects invalid UTF-8, minChars less than 1, and text
// longer than MaxRepeatedSubstringRunes runes. It returns nil, nil when no
// substring qualifies.
func RepeatedSubstrings(text string, minChars int) ([]string, error) {
	if !utf8.ValidString(text) {
		return nil, fmt.Errorf("%w: text must be valid UTF-8", ErrInvalidRepeatInput)
	}
	if minChars < 1 {
		return nil, fmt.Errorf("%w: minChars=%d must be at least 1", ErrInvalidRepeatInput, minChars)
	}

	runeCount := utf8.RuneCountInString(text)
	if runeCount > MaxRepeatedSubstringRunes {
		return nil, fmt.Errorf(
			"%w: text has %d runes, limit is %d",
			ErrInvalidRepeatInput, runeCount, MaxRepeatedSubstringRunes,
		)
	}
	if minChars >= runeCount {
		return nil, nil
	}

	runes := []rune(text)
	sa := buildRepeatSuffixArray(runes)
	lcp := buildRepeatLCP(runes, sa)

	type candidate struct {
		start  int32
		length int32
	}
	candidates := make([]candidate, 0)
	leftChars := make([]int32, 0, 64)

	for i := 1; i < runeCount; {
		length := lcp[i]
		if length == 0 {
			i++
			continue
		}

		runStart := i
		for i+1 < runeCount && lcp[i+1] == length {
			i++
		}
		runEnd := i
		i++

		if int(length) < minChars {
			continue
		}
		if lcp[runStart-1] >= length {
			continue
		}
		if runEnd+1 < runeCount && lcp[runEnd+1] >= length {
			continue
		}

		leftChars = leftChars[:0]
		first := int32(runeCount)
		for k := runStart - 1; k <= runEnd; k++ {
			position := sa[k]
			if position < first {
				first = position
			}
			left := int32(-1)
			if position > 0 {
				left = int32(runes[position-1])
			}
			leftChars = append(leftChars, left)
		}
		distinct := true
		if len(leftChars) <= repeatPairwiseLimit {
			for k := 0; k < len(leftChars) && distinct; k++ {
				for j := k + 1; j < len(leftChars); j++ {
					if leftChars[k] == leftChars[j] {
						distinct = false
						break
					}
				}
			}
		} else {
			sort.Sort(repeatInt32Slice(leftChars))
			for k := 1; k < len(leftChars); k++ {
				if leftChars[k] == leftChars[k-1] {
					distinct = false
					break
				}
			}
		}
		if !distinct {
			continue
		}

		candidates = append(candidates, candidate{
			start:  first,
			length: length,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].length != candidates[j].length {
			return candidates[i].length > candidates[j].length
		}
		return candidates[i].start < candidates[j].start
	})

	runeToByte := make([]int32, runeCount+1)
	runeIndex := 0
	for byteIndex := range text {
		runeToByte[runeIndex] = int32(byteIndex)
		runeIndex++
	}
	runeToByte[runeCount] = int32(len(text))

	result := make([]string, len(candidates))
	for i, candidate := range candidates {
		start := int(runeToByte[candidate.start])
		end := int(runeToByte[candidate.start+candidate.length])
		result[i] = text[start:end]
	}
	return result, nil
}

type repeatRuneSlice []rune

func (s repeatRuneSlice) Len() int           { return len(s) }
func (s repeatRuneSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s repeatRuneSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type repeatInt32Slice []int32

func (s repeatInt32Slice) Len() int           { return len(s) }
func (s repeatInt32Slice) Less(i, j int) bool { return s[i] < s[j] }
func (s repeatInt32Slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// buildRepeatSuffixArray builds the suffix array of runes with cyclic-shift
// doubling and a virtual sentinel smaller than every rune.
func buildRepeatSuffixArray(runes []rune) []int32 {
	n := len(runes)
	m := n + 1

	sorted := make([]rune, n)
	copy(sorted, runes)
	sort.Sort(repeatRuneSlice(sorted))

	uniqueCount := 0
	for i, r := range sorted {
		if i == 0 || r != sorted[i-1] {
			sorted[uniqueCount] = r
			uniqueCount++
		}
	}
	unique := sorted[:uniqueCount]

	rank := make([]int32, m)
	var classes int32 = 1
	for i, r := range runes {
		class := int32(sort.Search(uniqueCount, func(j int) bool {
			return unique[j] >= r
		})) + 1
		rank[i] = class
		if class+1 > classes {
			classes = class + 1
		}
	}

	p := make([]int32, m)
	classCounts := make([]int32, m)
	classStart := make([]int32, m)
	classCounts[0] = 1
	for i := 0; i < n; i++ {
		classCounts[rank[i]]++
	}
	var sum int32
	for i := int32(0); i < classes; i++ {
		classStart[i] = sum
		sum += classCounts[i]
	}
	p[classStart[0]] = int32(n)
	classStart[0]++
	for i := 0; i < n; i++ {
		class := rank[i]
		p[classStart[class]] = int32(i)
		classStart[class]++
	}

	c := rank
	c[n] = 0
	shifted := make([]int32, m)
	nextClasses := make([]int32, m)

	for shift := 1; shift < m && classes < int32(m); shift <<= 1 {
		for i := 0; i < m; i++ {
			position := int(p[i]) - shift
			if position < 0 {
				position += m
			}
			shifted[i] = int32(position)
		}

		for i := int32(0); i < classes; i++ {
			classCounts[i] = 0
		}
		for _, position := range shifted {
			classCounts[c[position]]++
		}
		sum = 0
		for i := int32(0); i < classes; i++ {
			classStart[i] = sum
			sum += classCounts[i]
		}
		for _, position := range shifted {
			class := c[position]
			p[classStart[class]] = position
			classStart[class]++
		}

		nextClasses[p[0]] = 0
		var currentClasses int32 = 1
		for i := 1; i < m; i++ {
			previous := p[i-1]
			current := p[i]
			previousNext := (int(previous) + shift) % m
			currentNext := (int(current) + shift) % m
			if c[previous] != c[current] || c[previousNext] != c[currentNext] {
				currentClasses++
			}
			nextClasses[current] = currentClasses - 1
		}
		c, nextClasses = nextClasses, c
		classes = currentClasses
	}

	return p[1:]
}

// buildRepeatLCP returns lcp where lcp[0] is 0 and lcp[i] is the longest
// common prefix of sa[i-1] and sa[i].
func buildRepeatLCP(runes []rune, sa []int32) []int32 {
	n := len(runes)
	rank := make([]int32, n)
	for i, position := range sa {
		rank[position] = int32(i)
	}

	lcp := make([]int32, n)
	height := 0
	for i := 0; i < n; i++ {
		position := rank[i]
		if position == 0 {
			height = 0
			continue
		}
		previous := int(sa[position-1])
		for i+height < n && previous+height < n && runes[i+height] == runes[previous+height] {
			height++
		}
		lcp[position] = int32(height)
		if height > 0 {
			height--
		}
	}
	return lcp
}
