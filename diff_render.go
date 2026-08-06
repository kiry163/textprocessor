package textprocessor

import (
	"errors"
	"fmt"
	"html"
	"strings"
)

const diffHTMLBaseCSS = `
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  color: var(--tp-text);
  background: var(--tp-page);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.tp-diff-page {
  width: min(calc(100% - 32px), 1080px);
  margin: 32px auto;
}
.tp-diff {
  margin: 0;
  padding: 20px 24px;
  color: var(--tp-text);
  background: var(--tp-surface);
  border: 1px solid var(--tp-border);
  border-radius: 6px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font: inherit;
  line-height: 1.7;
  tab-size: 4;
}
.tp-diff-insert,
.tp-diff-delete {
  padding: 0 0.08em;
  border-radius: 2px;
  box-decoration-break: clone;
  -webkit-box-decoration-break: clone;
}
.tp-diff-insert {
  color: var(--tp-insert-text);
  background: var(--tp-insert-bg);
  border-bottom: 2px solid var(--tp-insert-line);
  text-decoration: none;
}
.tp-diff-delete {
  color: var(--tp-delete-text);
  background: var(--tp-delete-bg);
  text-decoration-color: var(--tp-delete-line);
  text-decoration-thickness: 1.5px;
}
@media (max-width: 640px) {
  .tp-diff-page {
    width: 100%;
    margin: 0;
  }
  .tp-diff {
    padding: 16px;
    border-width: 0;
    border-radius: 0;
  }
}`

// DiffHTMLLightCSS is the built-in light theme.
const DiffHTMLLightCSS = `:root {
  color-scheme: light;
  --tp-page: #f7f7f8;
  --tp-surface: #ffffff;
  --tp-border: #d7d7db;
  --tp-text: #202124;
  --tp-insert-text: #166534;
  --tp-insert-bg: #dcfce7;
  --tp-insert-line: #22c55e;
  --tp-delete-text: #991b1b;
  --tp-delete-bg: #fee2e2;
  --tp-delete-line: #dc2626;
}` + diffHTMLBaseCSS

// DiffHTMLDarkCSS is the built-in dark theme.
const DiffHTMLDarkCSS = `:root {
  color-scheme: dark;
  --tp-page: #171717;
  --tp-surface: #222222;
  --tp-border: #4a4a4a;
  --tp-text: #f4f4f5;
  --tp-insert-text: #bbf7d0;
  --tp-insert-bg: #14532d;
  --tp-insert-line: #4ade80;
  --tp-delete-text: #fecaca;
  --tp-delete-bg: #7f1d1d;
  --tp-delete-line: #f87171;
}` + diffHTMLBaseCSS

// DiffHTMLHighContrastCSS is the built-in high-contrast theme.
const DiffHTMLHighContrastCSS = `:root {
  color-scheme: light;
  --tp-page: #ffffff;
  --tp-surface: #ffffff;
  --tp-border: #000000;
  --tp-text: #000000;
  --tp-insert-text: #000000;
  --tp-insert-bg: #7fffd4;
  --tp-insert-line: #006400;
  --tp-delete-text: #000000;
  --tp-delete-bg: #ffff00;
  --tp-delete-line: #b00000;
}` + diffHTMLBaseCSS

// DefaultDiffHTMLCSS is the default light theme.
const DefaultDiffHTMLCSS = DiffHTMLLightCSS

// DiffHTMLTheme selects a built-in stylesheet for a complete HTML document.
type DiffHTMLTheme string

const (
	DiffHTMLThemeLight        DiffHTMLTheme = "light"
	DiffHTMLThemeDark         DiffHTMLTheme = "dark"
	DiffHTMLThemeHighContrast DiffHTMLTheme = "high_contrast"
)

// DiffHTMLStyle configures the CSS classes used by RenderDiffHTML.
// When omitted, the renderer uses classes prefixed with "tp-diff".
type DiffHTMLStyle struct {
	ContainerClass string
	EqualClass     string
	InsertClass    string
	DeleteClass    string
}

// DiffHTMLDocumentOptions configures RenderDiffHTMLDocument.
// AdditionalCSS is appended after the selected theme so it can override it.
type DiffHTMLDocumentOptions struct {
	Title         string
	Language      string
	Theme         DiffHTMLTheme
	AdditionalCSS string
}

// DiffTextStyle configures the markers used by RenderDiffText.
// When omitted, insertions are green and deletions are red using ANSI codes.
type DiffTextStyle struct {
	EqualPrefix  string
	EqualSuffix  string
	InsertPrefix string
	InsertSuffix string
	DeletePrefix string
	DeleteSuffix string
	OmitEqual    bool
}

var (
	// ErrInvalidDiffPart indicates an unsupported operation in rendered input.
	ErrInvalidDiffPart = errors.New("invalid diff part")
	// ErrInvalidDiffHTMLTheme indicates an unsupported built-in HTML theme.
	ErrInvalidDiffHTMLTheme = errors.New("invalid diff HTML theme")
)

// RenderDiffHTML renders parts as an escaped HTML fragment inside a pre element.
func RenderDiffHTML(parts []DiffPart, styles ...DiffHTMLStyle) (string, error) {
	style := DiffHTMLStyle{
		ContainerClass: "tp-diff",
		EqualClass:     "tp-diff-equal",
		InsertClass:    "tp-diff-insert",
		DeleteClass:    "tp-diff-delete",
	}
	if len(styles) > 0 {
		style = styles[0]
	}

	var output strings.Builder
	output.WriteString("<pre")
	writeHTMLClass(&output, style.ContainerClass)
	output.WriteString(">")
	for _, part := range parts {
		tag, class, err := diffHTMLTag(part.Operation, style)
		if err != nil {
			return "", err
		}
		output.WriteByte('<')
		output.WriteString(tag)
		writeHTMLClass(&output, class)
		output.WriteByte('>')
		output.WriteString(html.EscapeString(part.Text))
		output.WriteString("</")
		output.WriteString(tag)
		output.WriteByte('>')
	}
	output.WriteString("</pre>")
	return output.String(), nil
}

// RenderDiffHTMLDocument renders parts as a complete, UTF-8 HTML5 document.
func RenderDiffHTMLDocument(parts []DiffPart, opts ...DiffHTMLDocumentOptions) (string, error) {
	options := DiffHTMLDocumentOptions{
		Title:    "Text Difference",
		Language: "en",
		Theme:    DiffHTMLThemeLight,
	}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.Title == "" {
		options.Title = "Text Difference"
	}
	if options.Language == "" {
		options.Language = "en"
	}
	if options.Theme == "" {
		options.Theme = DiffHTMLThemeLight
	}

	stylesheet, err := DiffHTMLThemeCSS(options.Theme)
	if err != nil {
		return "", err
	}
	fragment, err := RenderDiffHTML(parts)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	output.WriteString("<!doctype html>\n<html lang=\"")
	output.WriteString(html.EscapeString(options.Language))
	output.WriteString("\">\n<head>\n<meta charset=\"utf-8\">\n")
	output.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<title>")
	output.WriteString(html.EscapeString(options.Title))
	output.WriteString("</title>\n<style>\n")
	output.WriteString(stylesheet)
	if options.AdditionalCSS != "" {
		output.WriteByte('\n')
		output.WriteString(escapeStyleEndTag(options.AdditionalCSS))
	}
	output.WriteString("\n</style>\n</head>\n<body>\n<main class=\"tp-diff-page\">")
	output.WriteString(fragment)
	output.WriteString("</main>\n</body>\n</html>\n")
	return output.String(), nil
}

// DiffHTMLThemeCSS returns the stylesheet for a built-in HTML theme.
func DiffHTMLThemeCSS(theme DiffHTMLTheme) (string, error) {
	switch theme {
	case DiffHTMLThemeLight:
		return DiffHTMLLightCSS, nil
	case DiffHTMLThemeDark:
		return DiffHTMLDarkCSS, nil
	case DiffHTMLThemeHighContrast:
		return DiffHTMLHighContrastCSS, nil
	default:
		return "", fmt.Errorf("%w: theme=%q", ErrInvalidDiffHTMLTheme, theme)
	}
}

// RenderDiffText renders parts with ANSI colors by default or custom markers.
func RenderDiffText(parts []DiffPart, styles ...DiffTextStyle) (string, error) {
	style := DiffTextStyle{
		InsertPrefix: "\x1b[32m",
		InsertSuffix: "\x1b[0m",
		DeletePrefix: "\x1b[31m",
		DeleteSuffix: "\x1b[0m",
	}
	if len(styles) > 0 {
		style = styles[0]
	}

	var output strings.Builder
	for _, part := range parts {
		var prefix, suffix string
		switch part.Operation {
		case DiffEqual:
			if style.OmitEqual {
				continue
			}
			prefix, suffix = style.EqualPrefix, style.EqualSuffix
		case DiffInsert:
			prefix, suffix = style.InsertPrefix, style.InsertSuffix
		case DiffDelete:
			prefix, suffix = style.DeletePrefix, style.DeleteSuffix
		default:
			return "", fmt.Errorf("%w: operation=%q", ErrInvalidDiffPart, part.Operation)
		}
		output.WriteString(prefix)
		output.WriteString(part.Text)
		output.WriteString(suffix)
	}
	return output.String(), nil
}

func diffHTMLTag(operation DiffOperation, style DiffHTMLStyle) (tag, class string, err error) {
	switch operation {
	case DiffEqual:
		return "span", style.EqualClass, nil
	case DiffInsert:
		return "ins", style.InsertClass, nil
	case DiffDelete:
		return "del", style.DeleteClass, nil
	default:
		return "", "", fmt.Errorf("%w: operation=%q", ErrInvalidDiffPart, operation)
	}
}

func writeHTMLClass(output *strings.Builder, class string) {
	if class == "" {
		return
	}
	output.WriteString(` class="`)
	output.WriteString(html.EscapeString(class))
	output.WriteByte('"')
}

func escapeStyleEndTag(css string) string {
	lower := strings.ToLower(css)
	var output strings.Builder
	for {
		index := strings.Index(lower, "</style")
		if index < 0 {
			output.WriteString(css)
			return output.String()
		}
		output.WriteString(css[:index])
		output.WriteString(`\3c /style`)
		css = css[index+len("</style"):]
		lower = lower[index+len("</style"):]
	}
}
