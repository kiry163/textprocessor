package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kiry163/textprocessor"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "blocks" {
		return runBlocks(args[1:])
	}
	if len(args) > 0 && args[0] == "match" {
		return runMatch(args[1:])
	}
	return runSegment(args)
}

func runBlocks(args []string) error {
	fs := flag.NewFlagSet("textprocessor blocks", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	paragraphSplit := fs.String("paragraph-split", string(textprocessor.ParagraphSplitByLine), "paragraph split mode: line or blank-line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: textprocessor blocks [--paragraph-split line|blank-line] <file>")
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	splitMode, err := parseParagraphSplitMode(*paragraphSplit)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, block := range textprocessor.Blocks(string(data), textprocessor.BlockOptions{ParagraphSplit: splitMode}) {
		if err := encoder.Encode(block); err != nil {
			return err
		}
	}
	return nil
}

func parseParagraphSplitMode(mode string) (textprocessor.ParagraphSplitMode, error) {
	switch mode {
	case string(textprocessor.ParagraphSplitByLine):
		return textprocessor.ParagraphSplitByLine, nil
	case "blank-line", string(textprocessor.ParagraphSplitByBlankLine):
		return textprocessor.ParagraphSplitByBlankLine, nil
	default:
		return "", fmt.Errorf("invalid paragraph split mode %q: use line or blank-line", mode)
	}
}

func runSegment(args []string) error {
	fs := flag.NewFlagSet("textprocessor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	sentencesOnly := fs.Bool("sentences", false, "output one sentence per line")
	minChineseChars := fs.Int("min-chinese-chars", 50, "minimum Han characters per merged span; <=0 disables merging")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: textprocessor [--sentences] [--min-chinese-chars N] <file>")
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	text := string(data)

	if *sentencesOnly {
		for _, sentence := range textprocessor.Sentences(text) {
			fmt.Println(sentence)
		}
		return nil
	}

	encoder := json.NewEncoder(os.Stdout)
	for _, span := range textprocessor.Segment(text, textprocessor.SegmentOptions{MinChineseChars: *minChineseChars}) {
		if err := encoder.Encode(span); err != nil {
			return err
		}
	}
	return nil
}

func runMatch(args []string) error {
	fs := flag.NewFlagSet("textprocessor match", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mode := fs.String("mode", string(textprocessor.MatchSmart), "match mode: exact, normalized, or smart")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: textprocessor match [--mode exact|normalized|smart] <source-file> <query>")
	}

	source, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	query := fs.Arg(1)

	matchMode, err := parseMatchMode(*mode)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, match := range textprocessor.Match(string(source), query, textprocessor.MatchOptions{Mode: matchMode}) {
		if err := encoder.Encode(match); err != nil {
			return err
		}
	}
	return nil
}

func parseMatchMode(mode string) (textprocessor.MatchMode, error) {
	switch textprocessor.MatchMode(mode) {
	case textprocessor.MatchExact, textprocessor.MatchNormalized, textprocessor.MatchSmart:
		return textprocessor.MatchMode(mode), nil
	default:
		return "", fmt.Errorf("invalid match mode %q: use exact, normalized, or smart", mode)
	}
}
