package nlp

import (
	"math"
	"strings"
	"unicode"

	"github.com/raja-aiml/air/internal/engine"
)

// EmbeddingMatcher provides fast local matching using simple embeddings.
// This avoids API calls for common/clear commands.
type EmbeddingMatcher struct {
	commands   []*engine.Command
	vocabulary map[string]int
	vectors    map[string][]float64 // command name -> vector
}

// NewEmbeddingMatcher creates a new embedding matcher from commands.
func NewEmbeddingMatcher(commands []*engine.Command) *EmbeddingMatcher {
	m := &EmbeddingMatcher{
		commands:   commands,
		vocabulary: make(map[string]int),
		vectors:    make(map[string][]float64),
	}

	// Build vocabulary from all command names, descriptions, and examples
	vocabIndex := 0
	for _, cmd := range commands {
		for _, token := range tokenize(cmd.Name) {
			if _, exists := m.vocabulary[token]; !exists {
				m.vocabulary[token] = vocabIndex
				vocabIndex++
			}
		}
		for _, token := range tokenize(cmd.Description) {
			if _, exists := m.vocabulary[token]; !exists {
				m.vocabulary[token] = vocabIndex
				vocabIndex++
			}
		}
		for _, example := range cmd.Examples {
			for _, token := range tokenize(example) {
				if _, exists := m.vocabulary[token]; !exists {
					m.vocabulary[token] = vocabIndex
					vocabIndex++
				}
			}
		}
	}

	// Pre-compute vectors for each command (combining name, description, examples)
	for _, cmd := range commands {
		var allText []string
		allText = append(allText, tokenize(cmd.Name)...)
		allText = append(allText, tokenize(cmd.Description)...)
		for _, ex := range cmd.Examples {
			allText = append(allText, tokenize(ex)...)
		}
		m.vectors[cmd.Name] = m.vectorize(allText)
	}

	return m
}

// Match finds the best matching command for the input.
func (m *EmbeddingMatcher) Match(input string) (*ParseResult, error) {
	tokens := tokenize(input)
	inputVector := m.vectorize(tokens)

	var bestMatch string
	var bestScore float64

	for cmdName, cmdVector := range m.vectors {
		score := cosineSimilarity(inputVector, cmdVector)
		if score > bestScore {
			bestScore = score
			bestMatch = cmdName
		}
	}

	// Extract potential parameters from input
	params := m.extractParameters(input, bestMatch)

	return &ParseResult{
		Command:    bestMatch,
		Parameters: params,
		Confidence: bestScore,
		Source:     "embeddings",
		RawInput:   input,
	}, nil
}

// vectorize converts tokens to a TF vector.
func (m *EmbeddingMatcher) vectorize(tokens []string) []float64 {
	vector := make([]float64, len(m.vocabulary))

	// Count term frequencies
	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	// Build vector
	for token, count := range tf {
		if idx, exists := m.vocabulary[token]; exists {
			vector[idx] = float64(count)
		}
	}

	// Normalize
	var norm float64
	for _, v := range vector {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

// extractParameters attempts to extract parameter values from input.
func (m *EmbeddingMatcher) extractParameters(input, cmdName string) map[string]any {
	params := make(map[string]any)
	lower := strings.ToLower(input)

	// Find the command to get its parameter definitions
	var cmd *engine.Command
	for _, c := range m.commands {
		if c.Name == cmdName {
			cmd = c
			break
		}
	}
	if cmd == nil {
		return params
	}

	// Simple keyword-based parameter extraction
	for _, p := range cmd.Parameters {
		switch p.Type {
		case "bool":
			// Look for boolean indicators
			if containsAny(lower, []string{"detach", "background", "-d"}) && p.Name == "detached" {
				params[p.Name] = true
			}
			if containsAny(lower, []string{"volume", "remove volume", "-v"}) && p.Name == "removeVolumes" {
				params[p.Name] = true
			}
		case "string":
			// Look for service names
			services := []string{"postgres", "jaeger", "prometheus", "otel", "fluent"}
			for _, svc := range services {
				if strings.Contains(lower, svc) && p.Name == "service" {
					params[p.Name] = svc
					break
				}
			}
		}
	}

	return params
}

// tokenize splits text into normalized tokens.
func tokenize(text string) []string {
	text = strings.ToLower(text)

	// Split on non-alphanumeric characters
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	// Filter stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"to": true, "for": true, "of": true, "in": true, "on": true,
		"is": true, "are": true, "it": true, "this": true, "that": true,
		"me": true, "my": true, "i": true, "please": true, "can": true,
		"you": true, "want": true, "need": true, "would": true, "like": true,
	}

	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if !stopWords[t] && len(t) > 1 {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// containsAny checks if text contains any of the substrings.
func containsAny(text string, substrings []string) bool {
	for _, s := range substrings {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}
