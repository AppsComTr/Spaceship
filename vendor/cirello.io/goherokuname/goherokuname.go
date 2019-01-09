// Package goherokuname generates Heroku-like random names. Such as "black-block-0231".
//
// It supports both decimal and hexadecimal suffixes. It supports customized
// rendering, in which the delimiter, the length of suffix and the acceptable
// characters for suffix are tweakable.
package goherokuname // import "cirello.io/goherokuname"

import (
	"fmt"
	"math/rand"
)

// HaikunateCustom generates a Heroku-like random name in which the delimiter,
// length and acceptable characters for suffix are tweakable.
func HaikunateCustom(delimiter string, toklen int, tokchars string) string {
	noun := nouns[rand.Intn(len(nouns))]
	adjective := adjectives[rand.Intn(len(adjectives))]
	ret := adjective + delimiter + noun

	if toklen > 0 {
		token := ""
		for i := 0; i < toklen; i++ {
			token += string(tokchars[rand.Intn(len(tokchars))])
		}
		ret = ret + delimiter + token
	}

	return ret
}

// Haikunate generate standard Heroku-like random name, with "-" as delimiter,
// decimal with 4 digits.
func Haikunate() string {
	return HaikunateCustom("-", 4, "0123456789")
}

// HaikunateHex generate standard Heroku-like random name, with "-" as
// delimiter, hexadecimal with 4 digits.
func HaikunateHex() string {
	return HaikunateCustom("-", 4, "0123456789abcdef")
}

// Ubuntu generates a Ubuntu-like random name in which the delimiter is tweakable.
func Ubuntu(delimiter, letter string) (string, error) {
	if len(letter) == 0 {
		noun := nouns[rand.Intn(len(nouns))]
		letter = noun
	}
	chosenLetter := letter[0]

	adjIdx, ok := adjectivesIdx[chosenLetter]
	if !ok {
		return "", fmt.Errorf("could not find adjectives in dictionary starting with \"%c\"", chosenLetter)
	}

	nounIdx, ok := nounsIdx[chosenLetter]
	if !ok {
		return "", fmt.Errorf("could not find nouns in dictionary starting with \"%c\"", chosenLetter)
	}

	filteredAdjectives := adjectives[adjIdx[0] : adjIdx[1]+1]
	filteredNouns := nouns[nounIdx[0] : nounIdx[1]+1]

	adjective := filteredAdjectives[rand.Intn(len(filteredAdjectives))]
	noun := filteredNouns[rand.Intn(len(filteredNouns))]

	return adjective + delimiter + noun, nil
}
