package textprocessor_test

import (
	"reflect"
	"testing"

	"github.com/kiry163/textprocessor"
)

func TestBlocksSplitsParagraphsAndMarkdownPipeTable(t *testing.T) {
	input := "介绍段落。\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n结尾段落。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "介绍段落。", Start: 0, End: 5, Kind: textprocessor.BlockParagraph},
		{Text: "| A | B |\n|---|---|\n| 1 | 2 |", Start: 7, End: 36, Kind: textprocessor.BlockTable},
		{Text: "结尾段落。", Start: 38, End: 43, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksDefaultsToSplittingParagraphsByLine(t *testing.T) {
	input := "第一行。\n第二行。\n\n第三行。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "第一行。", Start: 0, End: 4, Kind: textprocessor.BlockParagraph},
		{Text: "第二行。", Start: 5, End: 9, Kind: textprocessor.BlockParagraph},
		{Text: "第三行。", Start: 11, End: 15, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksCanSplitParagraphsByBlankLine(t *testing.T) {
	input := "第一行。\n第二行。\n\n第三行。"

	got := textprocessor.Blocks(input, textprocessor.BlockOptions{ParagraphSplit: textprocessor.ParagraphSplitByBlankLine})
	want := []textprocessor.Block{
		{Text: "第一行。\n第二行。", Start: 0, End: 9, Kind: textprocessor.BlockParagraph},
		{Text: "第三行。", Start: 11, End: 15, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksKeepsDisplayMathAsFormulaBlocks(t *testing.T) {
	input := "前文。\n\n$$\nE = mc^2\n$$\n\n后文。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "前文。", Start: 0, End: 3, Kind: textprocessor.BlockParagraph},
		{Text: "$$\nE = mc^2\n$$", Start: 5, End: 19, Kind: textprocessor.BlockFormula},
		{Text: "后文。", Start: 21, End: 24, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksKeepsSingleLineDollarFormulaAsFormulaBlock(t *testing.T) {
	input := "$$ E = mc^2 $$\n下一段。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "$$ E = mc^2 $$", Start: 0, End: 14, Kind: textprocessor.BlockFormula},
		{Text: "下一段。", Start: 15, End: 19, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksKeepsBracketAndEquationEnvironmentsAsFormulaBlocks(t *testing.T) {
	input := "\\[\na+b=c\n\\]\n\n\\begin{equation}\nx=1\n\\end{equation}"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "\\[\na+b=c\n\\]", Start: 0, End: 11, Kind: textprocessor.BlockFormula},
		{Text: "\\begin{equation}\nx=1\n\\end{equation}", Start: 13, End: 48, Kind: textprocessor.BlockFormula},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksKeepsFencedCodeAsCodeBlock(t *testing.T) {
	input := "前文。\n```go\nfmt.Println(\"hi\")\n```\n后文。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "前文。", Start: 0, End: 3, Kind: textprocessor.BlockParagraph},
		{Text: "```go\nfmt.Println(\"hi\")\n```", Start: 4, End: 31, Kind: textprocessor.BlockCode},
		{Text: "后文。", Start: 32, End: 35, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksKeepsTildeFencedCodeAsCodeBlock(t *testing.T) {
	input := "~~~\nline\n~~~"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "~~~\nline\n~~~", Start: 0, End: 12, Kind: textprocessor.BlockCode},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}

func TestBlocksTreatsUnclosedFormulaAsParagraph(t *testing.T) {
	input := "$$\nE = mc^2\n普通文本。"

	got := textprocessor.Blocks(input)
	want := []textprocessor.Block{
		{Text: "$$", Start: 0, End: 2, Kind: textprocessor.BlockParagraph},
		{Text: "E = mc^2", Start: 3, End: 11, Kind: textprocessor.BlockParagraph},
		{Text: "普通文本。", Start: 12, End: 17, Kind: textprocessor.BlockParagraph},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Blocks() = %#v, want %#v", got, want)
	}
}
