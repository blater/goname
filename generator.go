// SPDX-License-Identifier: MIT

package goname

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

func validate(options Options) error {
	if options.Words < 1 {
		return errors.New("words must be a positive integer")
	}
	if options.MaxLetters < 0 {
		return errors.New("maxLetters must be zero or a positive integer")
	}
	if utf8.RuneCountInString(options.Separator) > 100 {
		return errors.New("separator must be 100 characters or less")
	}
	if options.Complexity > ComplexityLarge {
		return fmt.Errorf("invalid complexity: %d", options.Complexity)
	}
	return nil
}

func eligible(words []string, maxLetters int) ([]string, error) {
	if maxLetters == 0 {
		return words, nil
	}
	result := make([]string, 0, len(words))
	for _, word := range words {
		if utf8.RuneCountInString(word) <= maxLetters {
			result = append(result, word)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no words satisfy the maximum length of %d", maxLetters)
	}
	return result, nil
}

func (g *Generator) chooseInitial(categories [][]string) (rune, error) {
	var common map[rune]struct{}
	for _, category := range categories {
		initials := make(map[rune]struct{})
		for _, word := range category {
			initial, _ := utf8.DecodeRuneInString(word)
			initials[initial] = struct{}{}
		}
		if common == nil {
			common = initials
			continue
		}
		for initial := range common {
			if _, ok := initials[initial]; !ok {
				delete(common, initial)
			}
		}
	}
	if len(common) == 0 {
		return 0, errors.New("no alliterative goname can be generated from the eligible words")
	}

	initials := make([]rune, 0, len(common))
	for initial := range common {
		initials = append(initials, initial)
	}
	sort.Slice(initials, func(i, j int) bool { return initials[i] < initials[j] })

	g.mu.Lock()
	defer g.mu.Unlock()
	index, err := g.intn(len(initials))
	if err != nil {
		return 0, err
	}
	return initials[index], nil
}

func beginningWith(words []string, initial rune) []string {
	result := make([]string, 0, len(words))
	for _, word := range words {
		first, _ := utf8.DecodeRuneInString(word)
		if first == initial {
			result = append(result, word)
		}
	}
	return result
}

func join(words []string, separator string) string {
	return strings.Join(words, separator)
}
