# Trim Difference Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a rune-aware API that removes the exact common prefix and suffix from original and revised text, returns the changed original range, and optionally pads insertion-only changes with one context rune.

**Architecture:** Add the stable public API and its focused implementation in the root `textspan` package, following the existing one-capability-per-file organization. The algorithm operates on rune slices, returns a half-open original range, and keeps all behavior in `trim.go` until complexity justifies an `internal/diff` package.

**Tech Stack:** Go 1.18 standard library (`errors`, `fmt`), table-driven Go tests, existing root `textspan` package.

## Global Constraints

- Do not commit or push; the user has not authorized either operation.
- Keep the public API in package `textspan` at module path `github.com/kiry163/textprocessor`.
- Compare exact runes; do not normalize case, whitespace, punctuation, or Unicode.
- Report original offsets as a half-open rune range `[Start, End)`, never byte offsets.
- Default an empty original difference to left-context padding, falling back to right context at the beginning of the original.
- Do not add CLI support or reorganize unrelated code.
- Preserve compatibility with Go 1.18 and add no external dependencies.

---

## File Map

- Create `trim.go`: public result/options types, sentinel errors, and the rune-based trim algorithm.
- Create `trim_test.go`: external-package behavior, error, Unicode, and result-contract tests.
- Modify `README.md`: public usage, strategy, range semantics, and error documentation.

### Task 1: Public API and rune-based trim behavior

**Files:**
- Create: `trim.go`
- Create: `trim_test.go`

**Interfaces:**
- Consumes: Go strings `original` and `revised`, plus optional `TrimOptions`.
- Produces: `TrimDifference(original, revised string, opts ...TrimOptions) (TrimResult, error)`, `TrimResult`, `TrimOptions`, `TrimEmptyOriginalStrategy`, `ErrOriginalEmpty`, `ErrNoChange`, and `ErrInvalidTrimOptions`.

- [ ] **Step 1: Establish a clean test baseline**

Run:

```bash
go test ./... -count=1
```

Expected: all existing packages pass before the new tests are added.

- [ ] **Step 2: Write the complete failing public behavior tests**

Create `trim_test.go`:

```go
package textspan_test

import (
	"errors"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestTrimDifference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		revised  string
		opts     []textspan.TrimOptions
		want     textspan.TrimResult
	}{
		{
			name:     "replacement",
			original: "abc",
			revised:  "axc",
			want:     textspan.TrimResult{Original: "b", Revised: "x", Start: 1, End: 2},
		},
		{
			name:     "deletion",
			original: "abc",
			revised:  "ac",
			want:     textspan.TrimResult{Original: "b", Revised: "", Start: 1, End: 2},
		},
		{
			name:     "delete entire original",
			original: "abc",
			revised:  "",
			want:     textspan.TrimResult{Original: "abc", Revised: "", Start: 0, End: 3},
		},
		{
			name:     "complete replacement",
			original: "甲乙",
			revised:  "XY",
			want:     textspan.TrimResult{Original: "甲乙", Revised: "XY", Start: 0, End: 2},
		},
		{
			name:     "middle insertion explicitly pads left",
			original: "abc",
			revised:  "abXc",
			opts: []textspan.TrimOptions{{
				EmptyOriginal: textspan.TrimEmptyOriginalPad,
			}},
			want:     textspan.TrimResult{Original: "b", Revised: "bX", Start: 1, End: 2},
		},
		{
			name:     "start insertion falls back to right",
			original: "abc",
			revised:  "Xabc",
			want:     textspan.TrimResult{Original: "a", Revised: "Xa", Start: 0, End: 1},
		},
		{
			name:     "zero options use default padding",
			original: "abc",
			revised:  "abcX",
			opts:     []textspan.TrimOptions{{}},
			want:     textspan.TrimResult{Original: "c", Revised: "cX", Start: 2, End: 3},
		},
		{
			name:     "keep empty insertion range",
			original: "abc",
			revised:  "abXc",
			opts: []textspan.TrimOptions{{
				EmptyOriginal: textspan.TrimEmptyOriginalKeep,
			}},
			want: textspan.TrimResult{Original: "", Revised: "X", Start: 2, End: 2},
		},
		{
			name:     "emoji uses rune offsets",
			original: "你🙂好",
			revised:  "你🙂们好",
			want:     textspan.TrimResult{Original: "🙂", Revised: "🙂们", Start: 1, End: 2},
		},
		{
			name:     "punctuation and whitespace are compared exactly",
			original: "a， b",
			revised:  "a！ b",
			want:     textspan.TrimResult{Original: "，", Revised: "！", Start: 1, End: 2},
		},
		{
			name:     "one rune original supports start insertion",
			original: "a",
			revised:  "Xa",
			want:     textspan.TrimResult{Original: "a", Revised: "Xa", Start: 0, End: 1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := textspan.TrimDifference(tt.original, tt.revised, tt.opts...)
			if err != nil {
				t.Fatalf("TrimDifference() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("TrimDifference() = %#v, want %#v", got, tt.want)
			}

			originalRunes := []rune(tt.original)
			if got.Start < 0 || got.Start > got.End || got.End > len(originalRunes) {
				t.Fatalf("TrimDifference() range = [%d,%d), original rune length = %d", got.Start, got.End, len(originalRunes))
			}
			if fragment := string(originalRunes[got.Start:got.End]); fragment != got.Original {
				t.Fatalf("original range contains %q, result Original = %q", fragment, got.Original)
			}
			rebuilt := string(originalRunes[:got.Start]) + got.Revised + string(originalRunes[got.End:])
			if rebuilt != tt.revised {
				t.Fatalf("replacing result range rebuilt %q, want %q", rebuilt, tt.revised)
			}
		})
	}
}

func TestTrimDifferenceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		revised  string
		opts     []textspan.TrimOptions
		want     error
	}{
		{
			name:     "empty original",
			original: "",
			revised:  "X",
			want:     textspan.ErrOriginalEmpty,
		},
		{
			name:     "both inputs empty",
			original: "",
			revised:  "",
			want:     textspan.ErrOriginalEmpty,
		},
		{
			name:     "no change",
			original: "abc",
			revised:  "abc",
			want:     textspan.ErrNoChange,
		},
		{
			name:     "invalid strategy",
			original: "abc",
			revised:  "axc",
			opts: []textspan.TrimOptions{{
				EmptyOriginal: textspan.TrimEmptyOriginalStrategy("invalid"),
			}},
			want: textspan.ErrInvalidTrimOptions,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := textspan.TrimDifference(tt.original, tt.revised, tt.opts...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("TrimDifference() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run the tests and verify RED**

Run:

```bash
go test . -run '^TestTrimDifference' -count=1
```

Expected: compilation fails because `TrimOptions`, `TrimResult`, sentinel errors,
and `TrimDifference` are not defined. This is the expected RED state.

- [ ] **Step 4: Implement the minimal complete public behavior**

Create `trim.go`:

```go
package textspan

import (
	"errors"
	"fmt"
)

// TrimResult contains the differing text and its rune range in the original.
type TrimResult struct {
	Original string `json:"original"`
	Revised  string `json:"revised"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
}

// TrimEmptyOriginalStrategy controls insertion-only results.
type TrimEmptyOriginalStrategy string

const (
	// TrimEmptyOriginalPad includes one shared context rune in insertion-only results.
	TrimEmptyOriginalPad TrimEmptyOriginalStrategy = "pad"
	// TrimEmptyOriginalKeep preserves the empty original insertion range.
	TrimEmptyOriginalKeep TrimEmptyOriginalStrategy = "keep"
)

// TrimOptions configures TrimDifference.
type TrimOptions struct {
	EmptyOriginal TrimEmptyOriginalStrategy `json:"empty_original"`
}

var (
	// ErrOriginalEmpty indicates that original contains no runes.
	ErrOriginalEmpty = errors.New("original must not be empty")
	// ErrNoChange indicates that original and revised are identical.
	ErrNoChange = errors.New("original and revised have no change")
	// ErrInvalidTrimOptions indicates an unsupported option value.
	ErrInvalidTrimOptions = errors.New("invalid trim options")
)

// TrimDifference removes the exact common rune prefix and suffix from two texts.
func TrimDifference(original, revised string, opts ...TrimOptions) (TrimResult, error) {
	originalRunes := []rune(original)
	revisedRunes := []rune(revised)
	if len(originalRunes) == 0 {
		return TrimResult{}, ErrOriginalEmpty
	}
	if equalRunes(originalRunes, revisedRunes) {
		return TrimResult{}, ErrNoChange
	}

	strategy := TrimEmptyOriginalPad
	if len(opts) > 0 && opts[0].EmptyOriginal != "" {
		strategy = opts[0].EmptyOriginal
	}
	if strategy != TrimEmptyOriginalPad && strategy != TrimEmptyOriginalKeep {
		return TrimResult{}, fmt.Errorf("%w: empty_original=%q", ErrInvalidTrimOptions, strategy)
	}

	start := 0
	for start < len(originalRunes) && start < len(revisedRunes) && originalRunes[start] == revisedRunes[start] {
		start++
	}

	originalEnd := len(originalRunes)
	revisedEnd := len(revisedRunes)
	for originalEnd > start && revisedEnd > start && originalRunes[originalEnd-1] == revisedRunes[revisedEnd-1] {
		originalEnd--
		revisedEnd--
	}

	revisedStart := start
	if start == originalEnd && strategy == TrimEmptyOriginalPad {
		if start > 0 {
			start--
			revisedStart--
		} else {
			originalEnd++
			revisedEnd++
		}
	}

	return TrimResult{
		Original: string(originalRunes[start:originalEnd]),
		Revised:  string(revisedRunes[revisedStart:revisedEnd]),
		Start:    start,
		End:      originalEnd,
	}, nil
}

func equalRunes(left, right []rune) bool {
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
```

- [ ] **Step 5: Format the new Go files**

Run:

```bash
gofmt -w trim.go trim_test.go
```

Expected: command exits successfully and both files are formatted.

- [ ] **Step 6: Run the focused tests and verify GREEN**

Run:

```bash
go test . -run '^TestTrimDifference' -count=1
```

Expected: both `TestTrimDifference` and `TestTrimDifferenceErrors` pass.

- [ ] **Step 7: Run the full regression suite**

Run:

```bash
go test ./... -count=1
```

Expected: all root, CLI, processor, and replacer tests pass; packages without
tests report `[no test files]`.

### Task 2: Public documentation and final verification

**Files:**
- Modify: `README.md`, inserting the new section before `## CLI`.

**Interfaces:**
- Consumes: the public API completed in Task 1.
- Produces: user-facing examples for default padding, keep-empty behavior, rune offsets, and sentinel errors.

- [ ] **Step 1: Add README usage and behavior documentation**

Insert this section immediately before `## CLI` in `README.md`:

````markdown
## Trim Difference

`TrimDifference` removes the exact common rune prefix and suffix from an
original and revised string, then reports the changed range in the original:

```go
result, err := textspan.TrimDifference("abc", "abXc")
// result == textspan.TrimResult{
//     Original: "b",
//     Revised:  "bX",
//     Start:    1,
//     End:      2,
// }
```

`Start` and `End` are rune offsets in the original input. Replacing that
half-open range `[Start, End)` with `Revised` reconstructs the complete revised
input.

An insertion has an empty minimal original range. By default, the result
includes the shared rune immediately to its left; an insertion at the start
uses the shared rune to its right. This default is `TrimEmptyOriginalPad`.
Preserve the minimal empty range when needed:

```go
result, err := textspan.TrimDifference("abc", "abXc", textspan.TrimOptions{
    EmptyOriginal: textspan.TrimEmptyOriginalKeep,
})
// result == textspan.TrimResult{
//     Original: "",
//     Revised:  "X",
//     Start:    2,
//     End:      2,
// }
```

An empty original returns `ErrOriginalEmpty`. Identical inputs return
`ErrNoChange`. Invalid strategy values return an error matching
`ErrInvalidTrimOptions`; use `errors.Is` when branching on these errors.

````

- [ ] **Step 2: Verify formatting without changing unrelated files**

Run:

```bash
test -z "$(gofmt -l $(rg --files -g '*.go'))"
```

Expected: command exits successfully with no output. If it prints an existing
unrelated file, do not rewrite that file; report it separately.

- [ ] **Step 3: Run fresh full tests**

Run:

```bash
go test ./... -count=1
```

Expected: all tests pass with zero failures.

- [ ] **Step 4: Run static analysis**

Run:

```bash
go vet ./...
```

Expected: command exits successfully with no diagnostics.

- [ ] **Step 5: Verify the documented public surface is present**

Run:

```bash
rg -n 'TrimDifference|TrimEmptyOriginal(Pad|Keep)|Err(OriginalEmpty|NoChange|InvalidTrimOptions)' trim.go trim_test.go README.md
```

Expected: each documented API identifier appears in `trim.go`, is covered in
`trim_test.go`, and is described in `README.md`. Do not commit or push after
verification.
