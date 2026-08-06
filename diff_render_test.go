package textprocessor_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestRenderDiffHTMLEscapesTextAndIncludesOffsets(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.DiffPart{
		{Operation: textprocessor.DiffEqual, Text: "a", OriginalStart: 0, OriginalEnd: 1, RevisedStart: 0, RevisedEnd: 1},
		{Operation: textprocessor.DiffDelete, Text: "<old>", OriginalStart: 1, OriginalEnd: 6, RevisedStart: 1, RevisedEnd: 1},
		{Operation: textprocessor.DiffInsert, Text: "<script>x</script>", OriginalStart: 6, OriginalEnd: 6, RevisedStart: 1, RevisedEnd: 19},
	}

	got, err := textprocessor.RenderDiffHTML(parts)
	if err != nil {
		t.Fatalf("RenderDiffHTML() error = %v", err)
	}
	for _, fragment := range []string{
		`<pre class="tp-diff">`,
		`<del class="tp-diff-delete">&lt;old&gt;</del>`,
		`&lt;script&gt;x&lt;/script&gt;`,
		`</pre>`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("RenderDiffHTML() = %q, missing %q", got, fragment)
		}
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("RenderDiffHTML() emitted unescaped script tag: %q", got)
	}
	if strings.Contains(got, "data-original-") || strings.Contains(got, "data-revised-") {
		t.Fatalf("RenderDiffHTML() emitted offset attributes: %q", got)
	}
}

func TestRenderDiffHTMLCustomClasses(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.DiffPart{{Operation: textprocessor.DiffInsert, Text: "new"}}
	got, err := textprocessor.RenderDiffHTML(parts, textprocessor.DiffHTMLStyle{
		ContainerClass: `custom" container`,
		InsertClass:    "added",
	})
	if err != nil {
		t.Fatalf("RenderDiffHTML() error = %v", err)
	}
	if !strings.Contains(got, `class="custom&#34; container"`) || !strings.Contains(got, `<ins class="added"`) {
		t.Fatalf("RenderDiffHTML() = %q", got)
	}
}

func TestRenderDiffTextDefaultAndCustomStyles(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.DiffPart{
		{Operation: textprocessor.DiffEqual, Text: "same"},
		{Operation: textprocessor.DiffDelete, Text: "old"},
		{Operation: textprocessor.DiffInsert, Text: "new"},
	}

	got, err := textprocessor.RenderDiffText(parts)
	if err != nil {
		t.Fatalf("RenderDiffText() error = %v", err)
	}
	want := "same\x1b[31mold\x1b[0m\x1b[32mnew\x1b[0m"
	if got != want {
		t.Fatalf("RenderDiffText() = %q, want %q", got, want)
	}

	got, err = textprocessor.RenderDiffText(parts, textprocessor.DiffTextStyle{
		InsertPrefix: "{+",
		InsertSuffix: "+}",
		DeletePrefix: "[-",
		DeleteSuffix: "-]",
		OmitEqual:    true,
	})
	if err != nil {
		t.Fatalf("RenderDiffText(custom) error = %v", err)
	}
	if want := "[-old-]{+new+}"; got != want {
		t.Fatalf("RenderDiffText(custom) = %q, want %q", got, want)
	}
}

func TestDiffRenderersRejectUnknownOperation(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.DiffPart{{Operation: textprocessor.DiffOperation("unknown"), Text: "x"}}
	if _, err := textprocessor.RenderDiffHTML(parts); !errors.Is(err, textprocessor.ErrInvalidDiffPart) {
		t.Fatalf("RenderDiffHTML() error = %v, want ErrInvalidDiffPart", err)
	}
	if _, err := textprocessor.RenderDiffText(parts); !errors.Is(err, textprocessor.ErrInvalidDiffPart) {
		t.Fatalf("RenderDiffText() error = %v, want ErrInvalidDiffPart", err)
	}
}

func TestRenderDiffHTMLDocument(t *testing.T) {
	t.Parallel()

	parts := []textprocessor.DiffPart{{Operation: textprocessor.DiffInsert, Text: "新增"}}
	got, err := textprocessor.RenderDiffHTMLDocument(parts, textprocessor.DiffHTMLDocumentOptions{
		Title:         `A "diff"`,
		Language:      "zh-CN",
		Theme:         textprocessor.DiffHTMLThemeDark,
		AdditionalCSS: `.tp-diff { font-size: 18px; } </style><script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatalf("RenderDiffHTMLDocument() error = %v", err)
	}
	for _, fragment := range []string{
		"<!doctype html>",
		`<html lang="zh-CN">`,
		`<meta charset="utf-8">`,
		`<title>A &#34;diff&#34;</title>`,
		"color-scheme: dark",
		`<main class="tp-diff-page">`,
		`<ins class="tp-diff-insert">新增</ins>`,
		`\3c /style><script>alert(1)</script>`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("RenderDiffHTMLDocument() = %q, missing %q", got, fragment)
		}
	}
	if strings.Contains(got, "data-original-") || strings.Contains(got, "data-revised-") {
		t.Fatalf("RenderDiffHTMLDocument() emitted offset attributes: %q", got)
	}
}

func TestDiffHTMLThemeCSS(t *testing.T) {
	t.Parallel()

	for _, theme := range []textprocessor.DiffHTMLTheme{
		textprocessor.DiffHTMLThemeLight,
		textprocessor.DiffHTMLThemeDark,
		textprocessor.DiffHTMLThemeHighContrast,
	} {
		css, err := textprocessor.DiffHTMLThemeCSS(theme)
		if err != nil {
			t.Fatalf("DiffHTMLThemeCSS(%q) error = %v", theme, err)
		}
		if !strings.Contains(css, ".tp-diff-insert") || !strings.Contains(css, ".tp-diff-delete") {
			t.Fatalf("DiffHTMLThemeCSS(%q) = %q", theme, css)
		}
	}
	if _, err := textprocessor.DiffHTMLThemeCSS(textprocessor.DiffHTMLTheme("unknown")); !errors.Is(err, textprocessor.ErrInvalidDiffHTMLTheme) {
		t.Fatalf("DiffHTMLThemeCSS() error = %v, want ErrInvalidDiffHTMLTheme", err)
	}
}
