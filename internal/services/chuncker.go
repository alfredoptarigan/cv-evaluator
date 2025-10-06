package services

import (
	"strings"
	"unicode/utf8"
)

type TextChuncker interface {
	ChunkText(text string, maxChunkSize int, overlap int) []string
}

type textChunker struct{}

func NewTextChunker() TextChuncker {
	return &textChunker{}
}

// ChunkText implements TextChuncker.
func (tc *textChunker) ChunkText(text string, maxChunkSize int, overlap int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 1000
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxChunkSize {
		overlap = maxChunkSize / 4
	}

	// Split by paragraphs first
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If paragraph itself is too long, split by sentences
		if utf8.RuneCountInString(para) > maxChunkSize {
			sentences := splitIntoSentences(para)
			for _, sentence := range sentences {
				sentence = strings.TrimSpace(sentence)
				if sentence == "" {
					continue
				}

				// Check if adding this sentence would exceed max size
				if currentChunk.Len()+len(sentence)+1 > maxChunkSize {
					if currentChunk.Len() > 0 {
						chunks = append(chunks, currentChunk.String())

						// Add overlap from previous chunk
						currentChunk.Reset()
						if len(chunks) > 0 && overlap > 0 {
							prevChunk := chunks[len(chunks)-1]
							overlapText := getLastNChars(prevChunk, overlap)
							currentChunk.WriteString(overlapText)
							if overlapText != "" {
								currentChunk.WriteString(" ")
							}
						}
					}
				}

				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(sentence)
			}
		} else {
			// Check if adding this paragraph would exceed max size
			if currentChunk.Len()+len(para)+2 > maxChunkSize {
				if currentChunk.Len() > 0 {
					chunks = append(chunks, currentChunk.String())

					// Add overlap
					currentChunk.Reset()
					if len(chunks) > 0 && overlap > 0 {
						prevChunk := chunks[len(chunks)-1]
						overlapText := getLastNChars(prevChunk, overlap)
						currentChunk.WriteString(overlapText)
						if overlapText != "" {
							currentChunk.WriteString("\n\n")
						}
					}
				}
			}

			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
		}
	}

	// Add remaining chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

func splitIntoSentences(text string) []string {
	// Simple sentence splitter
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func getLastNChars(text string, n int) string {
	if n <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= n {
		return text
	}

	return string(runes[len(runes)-n:])
}
