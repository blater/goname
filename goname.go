// SPDX-License-Identifier: MIT

// Package goname generates human-readable random names from embedded or
// caller-supplied word lists.
package goname

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"sync"
)

// Complexity selects a dictionary tier.
type Complexity uint8

const (
	// ComplexityDefault selects the medium built-in dictionary, or the root of
	// a custom word directory.
	ComplexityDefault Complexity = iota
	ComplexitySmall
	ComplexityMedium
	ComplexityLarge
)

// Type selects a complete goname or one individual word category.
type Type uint8

const (
	TypeGoname Type = iota
	TypeAdverb
	TypeAdjective
	TypeName
)

// Options controls name generation. Start with DefaultOptions and change the
// fields required by the application.
type Options struct {
	// Words is the number of words in a complete goname.
	Words int

	// MaxLetters is the maximum number of Unicode code points per word. Zero
	// means unlimited.
	MaxLetters int

	// Separator is inserted between words. It may be empty.
	Separator string

	// Complexity selects a built-in dictionary tier, or a subdirectory tier
	// beneath WordDirectory.
	Complexity Complexity

	// Alliterate requires every generated word to begin with the same rune.
	Alliterate bool

	// Type selects a complete goname or one individual word category.
	Type Type

	// WordDirectory optionally names a directory containing adverbs.txt,
	// adjectives.txt, and names.txt.
	WordDirectory string
}

// DefaultOptions returns the standard two-word configuration.
func DefaultOptions() Options {
	return Options{
		Words:      2,
		Separator:  "-",
		Complexity: ComplexityDefault,
		Type:       TypeGoname,
	}
}

// Random is an application-controlled source of pseudorandom integers.
// math/rand.Rand implements Random. Calls are serialized by Generator.
type Random interface {
	Intn(n int) int
}

// Generator generates names using a configurable source of randomness. A
// Generator is safe for concurrent use.
type Generator struct {
	mu   sync.Mutex
	intn func(int) (int, error)
}

// NewGenerator returns a generator backed by cryptographically secure
// randomness.
func NewGenerator() *Generator {
	return &Generator{intn: func(n int) (int, error) {
		value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, fmt.Errorf("read secure randomness: %w", err)
		}
		return int(value.Int64()), nil
	}}
}

// NewSeededGenerator returns a deterministic generator for repeatable tests
// and workloads.
func NewSeededGenerator(seed int64) *Generator {
	return NewGeneratorWithRandom(mathrand.New(mathrand.NewSource(seed)))
}

// NewGeneratorWithRandom returns a generator backed by random. It panics when
// random is nil.
func NewGeneratorWithRandom(random Random) *Generator {
	if random == nil {
		panic("goname: nil Random")
	}
	return &Generator{intn: func(n int) (int, error) {
		return random.Intn(n), nil
	}}
}

var defaultGenerator = NewGenerator()

// Generate returns a goname using DefaultOptions.
func Generate() (string, error) {
	return defaultGenerator.Generate(DefaultOptions())
}

// GenerateWords returns a goname containing words words.
func GenerateWords(words int) (string, error) {
	options := DefaultOptions()
	options.Words = words
	return defaultGenerator.Generate(options)
}

// GenerateSeparated returns a goname with the requested word count and
// separator.
func GenerateSeparated(words int, separator string) (string, error) {
	options := DefaultOptions()
	options.Words = words
	options.Separator = separator
	return defaultGenerator.Generate(options)
}

// GenerateWithOptions returns a goname using options.
func GenerateWithOptions(options Options) (string, error) {
	return defaultGenerator.Generate(options)
}

// Generate returns a name using options.
func (g *Generator) Generate(options Options) (string, error) {
	if err := validate(options); err != nil {
		return "", err
	}
	words, err := loadWordLists(options)
	if err != nil {
		return "", err
	}

	switch options.Type {
	case TypeAdverb:
		return g.chooseEligible(words.adverbs, options.MaxLetters)
	case TypeAdjective:
		return g.chooseEligible(words.adjectives, options.MaxLetters)
	case TypeName:
		return g.chooseEligible(words.names, options.MaxLetters)
	case TypeGoname:
		return g.generateGoname(words, options)
	default:
		return "", fmt.Errorf("invalid name type: %d", options.Type)
	}
}

func (g *Generator) generateGoname(words wordLists, options Options) (string, error) {
	categories := make([][]string, 0, options.Words)
	if options.Words > 2 {
		adverbs, err := eligible(words.adverbs, options.MaxLetters)
		if err != nil {
			return "", err
		}
		for i := 2; i < options.Words; i++ {
			categories = append(categories, adverbs)
		}
	}
	if options.Words > 1 {
		adjectives, err := eligible(words.adjectives, options.MaxLetters)
		if err != nil {
			return "", err
		}
		categories = append(categories, adjectives)
	}
	names, err := eligible(words.names, options.MaxLetters)
	if err != nil {
		return "", err
	}
	categories = append(categories, names)

	var initial rune
	if options.Alliterate {
		initial, err = g.chooseInitial(categories)
		if err != nil {
			return "", err
		}
	}

	selected := make([]string, 0, len(categories))
	for _, category := range categories {
		candidates := category
		if options.Alliterate {
			candidates = beginningWith(category, initial)
		}
		word, chooseErr := g.choose(candidates)
		if chooseErr != nil {
			return "", chooseErr
		}
		selected = append(selected, word)
	}
	return join(selected, options.Separator), nil
}

func (g *Generator) chooseEligible(words []string, maxLetters int) (string, error) {
	candidates, err := eligible(words, maxLetters)
	if err != nil {
		return "", err
	}
	return g.choose(candidates)
}

func (g *Generator) choose(candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", errors.New("no eligible words")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	index, err := g.intn(len(candidates))
	if err != nil {
		return "", err
	}
	return candidates[index], nil
}
