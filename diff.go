package textprocessor

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	diffengine "github.com/kiry163/textprocessor/internal/diff"
)

// DiffOperation identifies how a DiffPart transforms the original text.
type DiffOperation string

const (
	DiffEqual  DiffOperation = "equal"
	DiffInsert DiffOperation = "insert"
	DiffDelete DiffOperation = "delete"
)

// DiffCleanupMode controls how the raw diff is adjusted for readability.
type DiffCleanupMode string

const (
	// DiffCleanupSemantic is the default and favors human-readable boundaries.
	DiffCleanupSemantic DiffCleanupMode = "semantic"
	// DiffCleanupNone keeps the algorithm's merged result without semantic cleanup.
	DiffCleanupNone DiffCleanupMode = "none"
)

// DiffOptions configures Diff. The first options value is used when supplied.
type DiffOptions struct {
	Cleanup DiffCleanupMode `json:"cleanup"`
	Timeout time.Duration   `json:"-"`
}

// DiffPart is one operation in the transformation from original to revised.
// All offsets are half-open rune ranges.
type DiffPart struct {
	Operation     DiffOperation `json:"operation"`
	Text          string        `json:"text"`
	OriginalStart int           `json:"original_start"`
	OriginalEnd   int           `json:"original_end"`
	RevisedStart  int           `json:"revised_start"`
	RevisedEnd    int           `json:"revised_end"`
}

var (
	// ErrInvalidDiffInput indicates that an input string is not valid UTF-8.
	ErrInvalidDiffInput = errors.New("diff input must be valid UTF-8")
	// ErrInvalidDiffOptions indicates an unsupported cleanup mode or timeout.
	ErrInvalidDiffOptions = errors.New("invalid diff options")
)

// Diff computes a complete rune-aware edit sequence from original to revised.
// Equal, Delete, and Insert parts together reconstruct both input strings.
func Diff(original, revised string, opts ...DiffOptions) ([]DiffPart, error) {
	if !utf8.ValidString(original) {
		return nil, fmt.Errorf("%w: original", ErrInvalidDiffInput)
	}
	if !utf8.ValidString(revised) {
		return nil, fmt.Errorf("%w: revised", ErrInvalidDiffInput)
	}

	options := DiffOptions{Cleanup: DiffCleanupSemantic}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.Cleanup == "" {
		options.Cleanup = DiffCleanupSemantic
	}
	if options.Cleanup != DiffCleanupSemantic && options.Cleanup != DiffCleanupNone {
		return nil, fmt.Errorf("%w: cleanup=%q", ErrInvalidDiffOptions, options.Cleanup)
	}
	if options.Timeout < 0 {
		return nil, fmt.Errorf("%w: timeout=%s", ErrInvalidDiffOptions, options.Timeout)
	}

	raw := diffengine.Compute(original, revised, diffengine.Options{
		Timeout:         options.Timeout,
		SemanticCleanup: options.Cleanup == DiffCleanupSemantic,
	})

	parts := make([]DiffPart, 0, len(raw))
	originalPos := 0
	revisedPos := 0
	for _, item := range raw {
		n := utf8.RuneCountInString(item.Text)
		part := DiffPart{
			Text:          item.Text,
			OriginalStart: originalPos,
			OriginalEnd:   originalPos,
			RevisedStart:  revisedPos,
			RevisedEnd:    revisedPos,
		}
		switch item.Operation {
		case diffengine.Equal:
			part.Operation = DiffEqual
			part.OriginalEnd += n
			part.RevisedEnd += n
			originalPos += n
			revisedPos += n
		case diffengine.Insert:
			part.Operation = DiffInsert
			part.RevisedEnd += n
			revisedPos += n
		case diffengine.Delete:
			part.Operation = DiffDelete
			part.OriginalEnd += n
			originalPos += n
		default:
			return nil, fmt.Errorf("unknown diff operation %d", item.Operation)
		}
		parts = append(parts, part)
	}
	return parts, nil
}
