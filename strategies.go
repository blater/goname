// SPDX-License-Identifier: MIT

package goname

import (
	"fmt"
	"math/big"
	"strings"
)

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const maxULIDTimestamp = (1 << 48) - 1

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
	g.mu.Lock()
	defer g.mu.Unlock()

	nowMillis := g.now().UnixMilli()
	if nowMillis < 0 || nowMillis > maxULIDTimestamp {
		return "", fmt.Errorf("ULID timestamp is out of range: %d", nowMillis)
	}
	timestamp := uint64(nowMillis)
	var entropy [10]byte
	if g.hasLastULID && timestamp <= g.lastULIDTimestamp {
		timestamp = g.lastULIDTimestamp
		entropy = g.lastULIDEntropy
		if !incrementULIDEntropy(&entropy) {
			if timestamp == maxULIDTimestamp {
				return "", fmt.Errorf("ULID monotonic entropy overflow at maximum timestamp")
			}
			timestamp++
		}
	} else {
		for index := range entropy {
			value, err := g.intn(256)
			if err != nil {
				return "", fmt.Errorf("read ULID randomness: %w", err)
			}
			entropy[index] = byte(value)
		}
	}

	var data [16]byte
	encodedTimestamp := timestamp
	for index := 5; index >= 0; index-- {
		data[index] = byte(encodedTimestamp)
		encodedTimestamp >>= 8
	}
	copy(data[6:], entropy[:])
	g.hasLastULID = true
	g.lastULIDTimestamp = timestamp
	g.lastULIDEntropy = entropy

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

func incrementULIDEntropy(entropy *[10]byte) bool {
	for index := len(entropy) - 1; index >= 0; index-- {
		entropy[index]++
		if entropy[index] != 0 {
			return true
		}
	}
	return false
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
