// Package diff contains the internal adapter around the underlying diff engine.
// Public callers should use the root textprocessor Diff API instead.
package diff

import (
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type Operation uint8

const (
	Equal Operation = iota
	Insert
	Delete
)

type Part struct {
	Operation Operation
	Text      string
}

type Options struct {
	Timeout         time.Duration
	SemanticCleanup bool
}

func Compute(original, revised string, options Options) []Part {
	engine := diffmatchpatch.New()
	engine.DiffTimeout = options.Timeout

	diffs := engine.DiffMain(original, revised, true)
	if options.SemanticCleanup {
		diffs = engine.DiffCleanupSemantic(diffs)
	}

	parts := make([]Part, 0, len(diffs))
	for _, item := range diffs {
		operation := Equal
		switch item.Type {
		case diffmatchpatch.DiffInsert:
			operation = Insert
		case diffmatchpatch.DiffDelete:
			operation = Delete
		}

		if item.Text == "" {
			continue
		}
		if len(parts) > 0 && parts[len(parts)-1].Operation == operation {
			parts[len(parts)-1].Text += item.Text
			continue
		}
		parts = append(parts, Part{Operation: operation, Text: item.Text})
	}
	return parts
}
