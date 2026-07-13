package textspan

import (
	"github.com/kiry163/textprocessor/internal/lang"
	"github.com/kiry163/textprocessor/internal/processor"
	"github.com/kiry163/textprocessor/internal/replacer"
	"github.com/kiry163/textprocessor/internal/segmenter"
)

func newSentenceSegmenter() *segmenter.Segmenter {
	cfg := lang.Default()
	punctuationReplacer := replacer.NewPunctuationReplacer()
	betweenPunctuationReplacer := cfg.BetweenPunctuationReplacer
	if betweenPunctuationReplacer == nil {
		betweenPunctuationReplacer = replacer.NewBetweenPunctuation(punctuationReplacer)
	}
	segmenterParams := &segmenter.Params{
		Config: cfg,
		Processor: processor.NewProcessor(processor.Params{
			Lang:                       cfg,
			ListItemReplacer:           replacer.NewListItemReplacer(),
			AbbrReplacer:               replacer.NewAbbreviationReplacer(cfg),
			PunctuationReplacer:        &punctuationReplacer,
			BetweenPunctuationReplacer: betweenPunctuationReplacer,
		}),
	}
	return segmenter.NewSegmenter(segmenterParams)
}

// Sentences segments Chinese-English mixed text into sentences.
func Sentences(text string) []string {
	return newSentenceSegmenter().Segment(text)
}
