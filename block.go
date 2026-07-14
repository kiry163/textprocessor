package textprocessor

import "strings"

type BlockKind string

const (
	BlockParagraph BlockKind = "paragraph"
	BlockTable     BlockKind = "table"
	BlockFormula   BlockKind = "formula"
	BlockCode      BlockKind = "code"
)

type Block struct {
	Text  string    `json:"text"`
	Start int       `json:"start"`
	End   int       `json:"end"`
	Kind  BlockKind `json:"kind"`
}

type ParagraphSplitMode string

const (
	ParagraphSplitByLine      ParagraphSplitMode = "line"
	ParagraphSplitByBlankLine ParagraphSplitMode = "blank_line"
)

type BlockOptions struct {
	ParagraphSplit ParagraphSplitMode `json:"paragraph_split"`
}

type blockLine struct {
	text  string
	start int
	end   int
}

// Blocks splits Markdown-like text into structural blocks with rune offsets.
func Blocks(text string, opts ...BlockOptions) []Block {
	options := BlockOptions{ParagraphSplit: ParagraphSplitByLine}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.ParagraphSplit == "" {
		options.ParagraphSplit = ParagraphSplitByLine
	}

	lines := blockLines(text)
	blocks := make([]Block, 0)
	paragraphStart := -1

	flushParagraph := func(endLine int) {
		if paragraphStart < 0 {
			return
		}
		blocks = append(blocks, makeBlock(lines, paragraphStart, endLine, BlockParagraph))
		paragraphStart = -1
	}

	for i := 0; i < len(lines); {
		if isBlankBlockLine(lines[i]) {
			flushParagraph(i)
			i++
			continue
		}

		if end, ok := formulaBlockEnd(lines, i); ok {
			flushParagraph(i)
			blocks = append(blocks, makeBlock(lines, i, end, BlockFormula))
			i = end
			continue
		}

		if end, ok := codeBlockEnd(lines, i); ok {
			flushParagraph(i)
			blocks = append(blocks, makeBlock(lines, i, end, BlockCode))
			i = end
			continue
		}

		if end, ok := tableBlockEnd(lines, i); ok {
			flushParagraph(i)
			blocks = append(blocks, makeBlock(lines, i, end, BlockTable))
			i = end
			continue
		}

		if paragraphStart < 0 {
			paragraphStart = i
		}
		if options.ParagraphSplit == ParagraphSplitByLine {
			flushParagraph(i + 1)
		}
		i++
	}
	flushParagraph(len(lines))
	return blocks
}

func blockLines(text string) []blockLine {
	runes := []rune(text)
	lines := make([]blockLine, 0)
	start := 0
	for i, r := range runes {
		if r != '\n' {
			continue
		}
		lines = append(lines, blockLine{text: string(runes[start:i]), start: start, end: i})
		start = i + 1
	}
	if start <= len(runes) {
		lines = append(lines, blockLine{text: string(runes[start:]), start: start, end: len(runes)})
	}
	return lines
}

func makeBlock(lines []blockLine, startLine, endLine int, kind BlockKind) Block {
	start := lines[startLine].start
	end := lines[endLine-1].end
	parts := make([]string, 0, endLine-startLine)
	for _, line := range lines[startLine:endLine] {
		parts = append(parts, line.text)
	}
	return Block{
		Text:  strings.Join(parts, "\n"),
		Start: start,
		End:   end,
		Kind:  kind,
	}
}

func isBlankBlockLine(line blockLine) bool {
	return strings.TrimSpace(line.text) == ""
}

func formulaBlockEnd(lines []blockLine, start int) (int, bool) {
	trimmed := strings.TrimSpace(lines[start].text)
	switch {
	case strings.HasPrefix(trimmed, "$$"):
		if len(trimmed) > 2 && strings.HasSuffix(trimmed, "$$") {
			return start + 1, true
		}
		return findFormulaEnd(lines, start+1, "$$")
	case trimmed == `\[`:
		return findFormulaEnd(lines, start+1, `\]`)
	case trimmed == `\begin{equation}`:
		return findFormulaEnd(lines, start+1, `\end{equation}`)
	default:
		return 0, false
	}
}

func findFormulaEnd(lines []blockLine, start int, marker string) (int, bool) {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i].text) == marker {
			return i + 1, true
		}
	}
	return 0, false
}

func codeBlockEnd(lines []blockLine, start int) (int, bool) {
	fence, ok := codeFenceMarker(lines[start].text)
	if !ok {
		return 0, false
	}
	for i := start + 1; i < len(lines); i++ {
		if closingCodeFence(lines[i].text, fence) {
			return i + 1, true
		}
	}
	return 0, false
}

func codeFenceMarker(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "```"):
		return "```", true
	case strings.HasPrefix(trimmed, "~~~"):
		return "~~~", true
	default:
		return "", false
	}
}

func closingCodeFence(line, fence string) bool {
	return strings.TrimSpace(line) == fence
}

func tableBlockEnd(lines []blockLine, start int) (int, bool) {
	if start+1 >= len(lines) || !isPipeTableLine(lines[start].text) || !isTableSeparatorLine(lines[start+1].text) {
		return 0, false
	}
	end := start + 2
	for end < len(lines) && isPipeTableLine(lines[end].text) {
		end++
	}
	return end, true
}

func isPipeTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

func isTableSeparatorLine(line string) bool {
	if !isPipeTableLine(line) {
		return false
	}
	cells := strings.Split(strings.TrimSpace(line), "|")
	for _, cell := range cells[1 : len(cells)-1] {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != '-' && r != ':' {
				return false
			}
		}
		if !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}
