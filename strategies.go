// SPDX-License-Identifier: MIT

package goname

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

type strategyAdapter interface {
	defaultWords() int
	generate(*Generator, Options) (string, error)
}

type dictionaryAdapter struct{}

func (dictionaryAdapter) defaultWords() int { return 2 }

func (dictionaryAdapter) generate(g *Generator, options Options) (string, error) {
	return g.generateDictionary(options)
}

type tokenAdapter struct {
	alphabet string
	length   int
	ulid     bool
}

func (a tokenAdapter) defaultWords() int { return 1 }

func (a tokenAdapter) generate(g *Generator, options Options) (string, error) {
	words := make([]string, options.Words)
	length := a.length
	if !a.ulid && options.MaxLetters > 0 {
		length = options.MaxLetters
	}
	for index := range words {
		word, err := a.generateWord(g, length)
		if err != nil {
			return "", err
		}
		if options.MixedCase {
			word = strings.ToUpper(word)
		} else {
			word = strings.ToLower(word)
		}
		words[index] = word
	}
	return join(words, options.Separator), nil
}

func (a tokenAdapter) generateWord(g *Generator, length int) (string, error) {
	if a.ulid {
		return g.generateULID()
	}
	var word strings.Builder
	word.Grow(length)
	for range length {
		index, err := g.randomIndex(len(a.alphabet))
		if err != nil {
			return "", err
		}
		word.WriteByte(a.alphabet[index])
	}
	return word.String(), nil
}

func (g *Generator) generateULID() (string, error) {
	var data [16]byte
	timestamp := uint64(time.Now().UnixMilli())
	for index := 5; index >= 0; index-- {
		data[index] = byte(timestamp)
		timestamp >>= 8
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	for index := 6; index < len(data); index++ {
		value, err := g.intn(256)
		if err != nil {
			return "", fmt.Errorf("read ULID randomness: %w", err)
		}
		data[index] = byte(value)
	}

	// A ULID encodes 128 bits in 26 base32 characters, padding the top two
	// bits with zero. This makes the first character range from 0 through 7.
	value := new(big.Int).SetBytes(data[:])
	mask := big.NewInt(31)
	encoded := make([]byte, 26)
	for index := len(encoded) - 1; index >= 0; index-- {
		digit := new(big.Int).And(new(big.Int).Set(value), mask).Int64()
		encoded[index] = crockfordBase32[digit]
		value.Rsh(value, 5)
	}
	return string(encoded), nil
}

func strategyAdapterFor(strategy Strategy) strategyAdapter {
	switch strategy {
	case StrategyHex:
		return tokenAdapter{alphabet: "0123456789abcdef", length: 4}
	case StrategyBase32:
		return tokenAdapter{alphabet: crockfordBase32, length: 4}
	case StrategyULID:
		return tokenAdapter{ulid: true}
	default:
		return dictionaryAdapter{}
	}
}
