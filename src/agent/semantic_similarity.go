package agent

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// SemanticSimilarityCalculator 语义相似度计算器
// Calculates semantic similarity between texts using bag-of-words model
// Supports both Chinese and English text tokenization
// Validates: Requirements 5.7, 8.3
type SemanticSimilarityCalculator struct {
	threshold float64 // Similarity threshold for determining if texts are similar
}

// NewSemanticSimilarityCalculator 创建语义相似度计算器
// Creates a new SemanticSimilarityCalculator with the specified threshold
// Parameters:
//   - threshold: the similarity threshold (0.0 to 1.0) for determining if texts are similar
//
// Returns a new SemanticSimilarityCalculator instance
// Validates: Requirements 5.7
func NewSemanticSimilarityCalculator(threshold float64) *SemanticSimilarityCalculator {
	// Ensure threshold is within valid range
	if threshold < 0.0 {
		threshold = 0.0
	}
	if threshold > 1.0 {
		threshold = 1.0
	}
	
	return &SemanticSimilarityCalculator{
		threshold: threshold,
	}
}

// GetThreshold returns the current similarity threshold
func (s *SemanticSimilarityCalculator) GetThreshold() float64 {
	return s.threshold
}

// SetThreshold sets a new similarity threshold
// Parameters:
//   - threshold: the new threshold value (0.0 to 1.0)
func (s *SemanticSimilarityCalculator) SetThreshold(threshold float64) {
	if threshold < 0.0 {
		threshold = 0.0
	}
	if threshold > 1.0 {
		threshold = 1.0
	}
	s.threshold = threshold
}

// CalculateSimilarity 计算两个文本的语义相似度
// Calculates the semantic similarity between two texts using cosine similarity
// on bag-of-words vectors
// Parameters:
//   - text1: the first text to compare
//   - text2: the second text to compare
//
// Returns a similarity score between 0.0 (completely different) and 1.0 (identical)
// Validates: Requirements 5.7, 8.3
func (s *SemanticSimilarityCalculator) CalculateSimilarity(text1, text2 string) float64 {
	// Handle edge cases
	if text1 == "" && text2 == "" {
		return 1.0 // Two empty strings are identical
	}
	if text1 == "" || text2 == "" {
		return 0.0 // One empty string means no similarity
	}
	
	// Exact match check
	if text1 == text2 {
		return 1.0
	}
	
	// Tokenize both texts
	tokens1 := tokenize(text1)
	tokens2 := tokenize(text2)
	
	// Handle edge cases after tokenization
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}
	
	// Build vocabulary and term frequency vectors
	vocab := buildVocabulary(tokens1, tokens2)
	vec1 := buildTermFrequencyVector(tokens1, vocab)
	vec2 := buildTermFrequencyVector(tokens2, vocab)
	
	// Calculate cosine similarity
	return cosineSimilarity(vec1, vec2)
}

// CalculateJaccardSimilarity 计算Jaccard相似度
// Calculates the Jaccard similarity between two texts
// Jaccard similarity = |A ∩ B| / |A ∪ B|
// Parameters:
//   - text1: the first text to compare
//   - text2: the second text to compare
//
// Returns a similarity score between 0.0 (no overlap) and 1.0 (identical sets)
func (s *SemanticSimilarityCalculator) CalculateJaccardSimilarity(text1, text2 string) float64 {
	// Handle edge cases
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}
	
	// Exact match check
	if text1 == text2 {
		return 1.0
	}
	
	// Tokenize both texts
	tokens1 := tokenize(text1)
	tokens2 := tokenize(text2)
	
	// Handle edge cases after tokenization
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}
	
	// Convert to sets
	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}
	
	set2 := make(map[string]bool)
	for _, token := range tokens2 {
		set2[token] = true
	}
	
	// Calculate intersection size
	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}
	
	// Calculate union size
	union := len(set1)
	for token := range set2 {
		if !set1[token] {
			union++
		}
	}
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}

// GetEmbedding 获取文本嵌入向量
// Gets the embedding vector for a text using bag-of-words representation
// The embedding is a normalized term frequency vector
// Parameters:
//   - text: the text to embed
//
// Returns a slice of float64 representing the embedding vector
// Note: The vector dimensions correspond to unique tokens in the text
// Validates: Requirements 5.7
func (s *SemanticSimilarityCalculator) GetEmbedding(text string) []float64 {
	if text == "" {
		return []float64{}
	}
	
	// Tokenize the text
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return []float64{}
	}
	
	// Build vocabulary from tokens
	vocab := make(map[string]int)
	for _, token := range tokens {
		if _, exists := vocab[token]; !exists {
			vocab[token] = len(vocab)
		}
	}
	
	// Build term frequency vector
	vec := make([]float64, len(vocab))
	for _, token := range tokens {
		if idx, exists := vocab[token]; exists {
			vec[idx]++
		}
	}
	
	// Normalize the vector
	return normalizeVector(vec)
}

// GetEmbeddingWithVocab 获取文本嵌入向量（使用指定词汇表）
// Gets the embedding vector for a text using a specified vocabulary
// This is useful when comparing multiple texts with a consistent vocabulary
// Parameters:
//   - text: the text to embed
//   - vocab: the vocabulary mapping tokens to indices
//
// Returns a slice of float64 representing the embedding vector
func (s *SemanticSimilarityCalculator) GetEmbeddingWithVocab(text string, vocab map[string]int) []float64 {
	if text == "" || len(vocab) == 0 {
		return make([]float64, len(vocab))
	}
	
	// Tokenize the text
	tokens := tokenize(text)
	
	// Build term frequency vector
	vec := make([]float64, len(vocab))
	for _, token := range tokens {
		if idx, exists := vocab[token]; exists {
			vec[idx]++
		}
	}
	
	// Normalize the vector
	return normalizeVector(vec)
}

// IsSimilar 判断两个文本是否相似
// Determines if two texts are similar based on the configured threshold
// Parameters:
//   - text1: the first text to compare
//   - text2: the second text to compare
//
// Returns true if the similarity score is >= threshold, false otherwise
// Validates: Requirements 5.2
func (s *SemanticSimilarityCalculator) IsSimilar(text1, text2 string) bool {
	similarity := s.CalculateSimilarity(text1, text2)
	return similarity >= s.threshold
}

// IsSimilarWithJaccard 使用Jaccard相似度判断两个文本是否相似
// Determines if two texts are similar using Jaccard similarity
// Parameters:
//   - text1: the first text to compare
//   - text2: the second text to compare
//
// Returns true if the Jaccard similarity score is >= threshold, false otherwise
func (s *SemanticSimilarityCalculator) IsSimilarWithJaccard(text1, text2 string) bool {
	similarity := s.CalculateJaccardSimilarity(text1, text2)
	return similarity >= s.threshold
}

// tokenize 分词
// Tokenizes text into individual tokens, supporting both Chinese and English
// Chinese text is segmented character by character (unigram)
// English text is segmented by whitespace and punctuation
// All tokens are converted to lowercase for consistency
// Parameters:
//   - text: the text to tokenize
//
// Returns a slice of tokens
// Validates: Requirements 8.3
func tokenize(text string) []string {
	if text == "" {
		return []string{}
	}
	
	// Normalize text: convert to lowercase
	text = strings.ToLower(text)
	
	var tokens []string
	var currentWord strings.Builder
	
	for _, r := range text {
		if isChinese(r) {
			// Flush any accumulated English word
			if currentWord.Len() > 0 {
				word := currentWord.String()
				if isValidToken(word) {
					tokens = append(tokens, word)
				}
				currentWord.Reset()
			}
			// Add Chinese character as a token
			tokens = append(tokens, string(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// Accumulate English letters and digits
			currentWord.WriteRune(r)
		} else {
			// Whitespace or punctuation: flush current word
			if currentWord.Len() > 0 {
				word := currentWord.String()
				if isValidToken(word) {
					tokens = append(tokens, word)
				}
				currentWord.Reset()
			}
		}
	}
	
	// Flush any remaining word
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if isValidToken(word) {
			tokens = append(tokens, word)
		}
	}
	
	return tokens
}

// isChinese checks if a rune is a Chinese character
// Covers CJK Unified Ideographs (U+4E00 to U+9FFF)
// and CJK Unified Ideographs Extension A (U+3400 to U+4DBF)
func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0x20000 && r <= 0x2A6DF) // CJK Unified Ideographs Extension B
}

// isValidToken checks if a token is valid (not a stopword and has meaningful content)
func isValidToken(token string) bool {
	if len(token) == 0 {
		return false
	}
	
	// Filter out very short tokens (single letters except for Chinese)
	if len(token) == 1 && !isChinese(rune(token[0])) {
		return false
	}
	
	// Filter out common English stopwords
	stopwords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"of": true, "in": true, "to": true, "for": true, "with": true,
		"on": true, "at": true, "by": true, "from": true, "as": true,
		"and": true, "or": true, "but": true, "if": true, "then": true,
		"so": true, "than": true, "that": true, "this": true, "these": true,
		"those": true, "it": true, "its": true,
	}
	
	return !stopwords[token]
}

// buildVocabulary builds a vocabulary from two token lists
// Returns a map from token to index
func buildVocabulary(tokens1, tokens2 []string) map[string]int {
	vocab := make(map[string]int)
	
	for _, token := range tokens1 {
		if _, exists := vocab[token]; !exists {
			vocab[token] = len(vocab)
		}
	}
	
	for _, token := range tokens2 {
		if _, exists := vocab[token]; !exists {
			vocab[token] = len(vocab)
		}
	}
	
	return vocab
}

// buildTermFrequencyVector builds a term frequency vector for tokens
// using the provided vocabulary
func buildTermFrequencyVector(tokens []string, vocab map[string]int) []float64 {
	vec := make([]float64, len(vocab))
	
	for _, token := range tokens {
		if idx, exists := vocab[token]; exists {
			vec[idx]++
		}
	}
	
	return vec
}

// cosineSimilarity calculates the cosine similarity between two vectors
// cosine_similarity = (A · B) / (||A|| * ||B||)
func cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}
	
	if len(vec1) == 0 {
		return 1.0
	}
	
	// Calculate dot product and magnitudes
	var dotProduct float64
	var magnitude1 float64
	var magnitude2 float64
	
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		magnitude1 += vec1[i] * vec1[i]
		magnitude2 += vec2[i] * vec2[i]
	}
	
	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)
	
	// Handle zero magnitude (empty vectors)
	if magnitude1 == 0.0 || magnitude2 == 0.0 {
		if magnitude1 == 0.0 && magnitude2 == 0.0 {
			return 1.0
		}
		return 0.0
	}
	
	return dotProduct / (magnitude1 * magnitude2)
}

// normalizeVector normalizes a vector to unit length
func normalizeVector(vec []float64) []float64 {
	if len(vec) == 0 {
		return vec
	}
	
	// Calculate magnitude
	var magnitude float64
	for _, v := range vec {
		magnitude += v * v
	}
	magnitude = math.Sqrt(magnitude)
	
	// Handle zero magnitude
	if magnitude == 0.0 {
		return vec
	}
	
	// Normalize
	normalized := make([]float64, len(vec))
	for i, v := range vec {
		normalized[i] = v / magnitude
	}
	
	return normalized
}

// TokenizeText is an exported version of tokenize for external use
// Tokenizes text into individual tokens, supporting both Chinese and English
// Parameters:
//   - text: the text to tokenize
//
// Returns a slice of tokens
func TokenizeText(text string) []string {
	return tokenize(text)
}

// Common Chinese stopwords for more advanced filtering (optional use)
var chineseStopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "我": true,
	"有": true, "和": true, "就": true, "不": true, "人": true,
	"都": true, "一": true, "一个": true, "上": true, "也": true,
	"很": true, "到": true, "说": true, "要": true, "去": true,
	"你": true, "会": true, "着": true, "没有": true, "看": true,
	"好": true, "自己": true, "这": true, "那": true, "里": true,
}

// TokenizeWithStopwords tokenizes text and optionally filters stopwords
// Parameters:
//   - text: the text to tokenize
//   - filterStopwords: whether to filter out stopwords
//
// Returns a slice of tokens
func TokenizeWithStopwords(text string, filterStopwords bool) []string {
	tokens := tokenize(text)
	
	if !filterStopwords {
		return tokens
	}
	
	// Filter Chinese stopwords
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !chineseStopwords[token] {
			filtered = append(filtered, token)
		}
	}
	
	return filtered
}

// ExtractNGrams extracts n-grams from text
// Parameters:
//   - text: the text to extract n-grams from
//   - n: the size of n-grams (e.g., 2 for bigrams, 3 for trigrams)
//
// Returns a slice of n-gram strings
func ExtractNGrams(text string, n int) []string {
	if n <= 0 {
		return []string{}
	}
	
	tokens := tokenize(text)
	if len(tokens) < n {
		return []string{}
	}
	
	ngrams := make([]string, 0, len(tokens)-n+1)
	for i := 0; i <= len(tokens)-n; i++ {
		ngram := strings.Join(tokens[i:i+n], " ")
		ngrams = append(ngrams, ngram)
	}
	
	return ngrams
}

// Regular expression for extracting words (used as fallback)
var wordRegex = regexp.MustCompile(`[\p{L}\p{N}]+`)

// TokenizeSimple provides a simple tokenization using regex
// This is a fallback method that may be faster for simple cases
func TokenizeSimple(text string) []string {
	if text == "" {
		return []string{}
	}
	
	text = strings.ToLower(text)
	matches := wordRegex.FindAllString(text, -1)
	
	// Filter out stopwords
	filtered := make([]string, 0, len(matches))
	for _, match := range matches {
		if isValidToken(match) {
			filtered = append(filtered, match)
		}
	}
	
	return filtered
}
