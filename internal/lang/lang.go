package lang

import "github.com/kiry163/textprocessor/internal/processor"

func Default() *processor.Config {
	return newChinese()
}
