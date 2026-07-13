package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIOutputsSpanJSONLines(t *testing.T) {
	path := writeTempInput(t, "你好。Hello.")

	cmd := exec.Command("go", "run", ".", "--min-chinese-chars", "0", path)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "{\"text\":\"你好。\",\"start\":0,\"end\":3}\n{\"text\":\"Hello.\",\"start\":3,\"end\":9}\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestCLIOutputsSentences(t *testing.T) {
	path := writeTempInput(t, "你好。Hello.")

	cmd := exec.Command("go", "run", ".", "--sentences", path)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "你好。\nHello.\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestCLIBlocksOutputsBlockJSONLines(t *testing.T) {
	path := writeTempInput(t, "介绍段落。\n\n| A | B |\n|---|---|\n| 1 | 2 |")

	cmd := exec.Command("go", "run", ".", "blocks", path)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "{\"text\":\"介绍段落。\",\"start\":0,\"end\":5,\"kind\":\"paragraph\"}\n" +
		"{\"text\":\"| A | B |\\n|---|---|\\n| 1 | 2 |\",\"start\":7,\"end\":36,\"kind\":\"table\"}\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestCLIBlocksCanSplitParagraphsByBlankLine(t *testing.T) {
	path := writeTempInput(t, "第一行。\n第二行。\n\n第三行。")

	cmd := exec.Command("go", "run", ".", "blocks", "--paragraph-split", "blank-line", path)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "{\"text\":\"第一行。\\n第二行。\",\"start\":0,\"end\":9,\"kind\":\"paragraph\"}\n" +
		"{\"text\":\"第三行。\",\"start\":11,\"end\":15,\"kind\":\"paragraph\"}\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestCLIMatchOutputsExactMatches(t *testing.T) {
	sourcePath := writeTempInput(t, "前缀 ABC 后缀 ABC")

	cmd := exec.Command("go", "run", ".", "match", "--mode", "exact", sourcePath, "ABC")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "{\"text\":\"ABC\",\"start\":3,\"end\":6}\n{\"text\":\"ABC\",\"start\":10,\"end\":13}\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestCLIMatchOutputsNormalizedMatches(t *testing.T) {
	sourcePath := writeTempInput(t, "他说：Hello, world！")

	cmd := exec.Command("go", "run", ".", "match", "--mode", "normalized", sourcePath, "Hello world")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, out)
	}

	want := "{\"text\":\"Hello, world\",\"start\":3,\"end\":15}\n"
	if string(out) != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func writeTempInput(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write temp input: %v", err)
	}
	return path
}
