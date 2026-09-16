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
	"time"
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

// Strategy selects a dictionary or token generation adapter independently of
// the standard complexity tiers.
type Strategy uint8

const (
	// StrategyDefault uses the selected small, medium, or large word lists.
	StrategyDefault Strategy = iota
	// StrategyTolkien uses Tolkien character and place names with themed modifiers.
	StrategyTolkien
	// StrategyHex generates four-character hexadecimal tokens.
	StrategyHex
	// StrategyBase32 generates four-character Crockford Base32 tokens.
	StrategyBase32
	// StrategyULID generates 26-character Universally Unique Lexicographically
	// Sortable Identifiers, monotonic within a Generator instance.
	StrategyULID
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
	// Words is the number of words in a complete goname. Zero selects the
	// strategy's default: two for dictionary strategies and one for token
	// strategies.
	Words int

	// MaxLetters is the maximum number of Unicode code points per dictionary
	// word. For hex and Base32 strategies, a positive value sets token width;
	// ULIDs retain their fixed width. Zero selects each strategy's normal width.
	MaxLetters int

	// Separator is inserted between words. It may be empty.
	Separator string

	// Prefix is prepended to the generated name, followed by Separator. It does
	// not count toward Words.
	Prefix string

	// Complexity selects a built-in dictionary tier, or a matching subdirectory
	// beneath WordDirectory. It cannot be combined with a non-default Strategy.
	Complexity Complexity

	// Strategy selects a word-list or token-generation adapter. It cannot be
	// combined with a non-default Complexity.
	Strategy Strategy

	// Alliterate requires every dictionary word to begin with the same letter,
	// ignoring case.
	Alliterate bool

	// MixedCase preserves dictionary word casing and emits token strategies in
	// uppercase. When false, generated words and tokens are lowercase.
	MixedCase bool

	// Type selects a complete goname or one individual word category.
	Type Type

	// WordDirectory optionally names a directory containing the word lists, or
	// subdirectories named for a selected complexity tier or strategy.
	WordDirectory string
}

// DefaultOptions returns the standard configuration. The default word count
// is selected by the strategy: two for dictionary strategies and one for
// token strategies.
func DefaultOptions() Options {
	return Options{
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
// Generator is safe for concurrent use, and its ULIDs are monotonic within
// that Generator instance.
type Generator struct {
	mu                sync.Mutex
	intn              func(int) (int, error)
	now               func() time.Time
	hasLastULID       bool
	lastULIDTimestamp uint64
	lastULIDEntropy   [10]byte
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
	}, now: time.Now}
}

// NewSeededGenerator returns a generator with a repeatable pseudorandom
// sequence. ULID timestamps still use the current time, and ULID monotonic
// state is local to this generator.
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
	}, now: time.Now}
}

var defaultGenerator = NewGenerator()

// Generate returns a goname using DefaultOptions.
func Generate() (string, error) {
	return defaultGenerator.Generate(DefaultOptions())
}

// GenerateWords returns a goname containing words words.
func GenerateWords(words int) (string, error) {
	return GenerateSeparated(words, DefaultOptions().Separator)
}

// GenerateSeparated returns a goname with the requested word count and
// separator.
func GenerateSeparated(words int, separator string) (string, error) {
	if words < 1 {
		return "", errors.New("words must be a positive integer")
	}
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
	adapter := strategyAdapterFor(options.Strategy)
	if options.Words == 0 {
		options.Words = adapter.defaultWords()
	}
	name, err := adapter.generate(g, options)
	if err != nil {
		return "", err
	}
	if options.Prefix != "" {
		name = options.Prefix + options.Separator + name
	}
	return name, nil
}

func (g *Generator) generateDictionary(options Options) (string, error) {
	words, err := loadWordLists(options)
	if err != nil {
		return "", err
	}

	switch options.Type {
	case TypeAdverb:
		word, err := g.chooseEligible(words.adverbs, options.MaxLetters)
		return applyCase(word, options.MixedCase), err
	case TypeAdjective:
		word, err := g.chooseEligible(words.adjectives, options.MaxLetters)
		return applyCase(word, options.MixedCase), err
	case TypeName:
		word, err := g.chooseEligible(words.names, options.MaxLetters)
		return applyCase(word, options.MixedCase), err
	case TypeGoname:
		return g.generateGoname(words, options)
	default:
		return "", fmt.Errorf("invalid name type: %d", options.Type)
	}
}

// gonameCategories builds the grammar in selection order, filtering each
// category once even when multiple adverbs are requested.
func gonameCategories(words wordLists, options Options) ([][]string, error) {
	categories := make([][]string, 0, options.Words)
	for _, category := range []struct {
		words []string
		count int
	}{
		{words.adverbs, max(0, options.Words-2)},
		{words.adjectives, min(1, max(0, options.Words-1))},
		{words.names, 1},
	} {
		if category.count == 0 {
			continue
		}
		candidates, err := eligible(category.words, options.MaxLetters)
		if err != nil {
			return nil, err
		}
		for i := 0; i < category.count; i++ {
			categories = append(categories, candidates)
		}
	}
	return categories, nil
}

func (g *Generator) generateGoname(words wordLists, options Options) (string, error) {
	categories, err := gonameCategories(words, options)
	if err != nil {
		return "", err
	}

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
		selected = append(selected, applyCase(word, options.MixedCase))
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
	index, err := g.randomIndex(len(candidates))
	if err != nil {
		return "", err
	}
	return candidates[index], nil
}

func (g *Generator) randomIndex(n int) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	index, err := g.intn(n)
	if err != nil {
		return 0, err
	}
	return index, nil
}
