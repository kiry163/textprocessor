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
	fs := flag.NewFlagSet("textspan blocks", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	paragraphSplit := fs.String("paragraph-split", string(textspan.ParagraphSplitByLine), "paragraph split mode: line or blank-line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: textspan blocks [--paragraph-split line|blank-line] <file>")
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
	for _, block := range textspan.Blocks(string(data), textspan.BlockOptions{ParagraphSplit: splitMode}) {
		if err := encoder.Encode(block); err != nil {
			return err
		}
	}
	return nil
}

func parseParagraphSplitMode(mode string) (textspan.ParagraphSplitMode, error) {
	switch mode {
	case string(textspan.ParagraphSplitByLine):
		return textspan.ParagraphSplitByLine, nil
	case "blank-line", string(textspan.ParagraphSplitByBlankLine):
		return textspan.ParagraphSplitByBlankLine, nil
	default:
		return "", fmt.Errorf("invalid paragraph split mode %q: use line or blank-line", mode)
	}
}

func runSegment(args []string) error {
	fs := flag.NewFlagSet("textspan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	sentencesOnly := fs.Bool("sentences", false, "output one sentence per line")
	minChineseChars := fs.Int("min-chinese-chars", 50, "minimum Han characters per merged span; <=0 disables merging")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: textspan [--sentences] [--min-chinese-chars N] <file>")
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	text := string(data)

	if *sentencesOnly {
		for _, sentence := range textspan.Sentences(text) {
			fmt.Println(sentence)
		}
		return nil
	}

	encoder := json.NewEncoder(os.Stdout)
	for _, span := range textspan.Segment(text, textspan.SegmentOptions{MinChineseChars: *minChineseChars}) {
		if err := encoder.Encode(span); err != nil {
			return err
		}
	}
	return nil
}

func runMatch(args []string) error {
	fs := flag.NewFlagSet("textspan match", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mode := fs.String("mode", string(textspan.MatchSmart), "match mode: exact, normalized, or smart")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: textspan match [--mode exact|normalized|smart] <source-file> <query>")
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
	for _, match := range textspan.Match(string(source), query, textspan.MatchOptions{Mode: matchMode}) {
		if err := encoder.Encode(match); err != nil {
			return err
		}
	}
	return nil
}

func parseMatchMode(mode string) (textspan.MatchMode, error) {
	switch textspan.MatchMode(mode) {
	case textspan.MatchExact, textspan.MatchNormalized, textspan.MatchSmart:
		return textspan.MatchMode(mode), nil
	default:
		return "", fmt.Errorf("invalid match mode %q: use exact, normalized, or smart", mode)
	}
}
