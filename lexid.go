package lexid

import (
	"errors"
	"fmt"
	"sort"
)

const (
	// CharsAll contains all visible ASCII characters
	CharsAll = "!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

	// CharsAllNoEscape contains all visible ASCII characters except those that need to be escaped in JSON (", \)
	CharsAllNoEscape = "!#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"

	// CharsAlphanumeric contains alphanumeric characters (uppercase and lowercase letters, and digits)
	CharsAlphanumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	// CharsAlphanumericLower contains alphanumeric lowercase characters (lowercase letters and digits)
	CharsAlphanumericLower = "abcdefghijklmnopqrstuvwxyz0123456789"

	// CharsBase64 contains the Base64 character set (URL-safe)
	CharsBase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	// CharsBase58 contains the Base58 character set (no 0, O, I, l)
	CharsBase58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

// Must creates a Lexid and panics if there is an error
func Must(chars string, blockSize, stepSize int) *Lexid {
	lexid, err := New(chars, blockSize, stepSize)
	if err != nil {
		panic(err)
	}
	return lexid
}

// New creates a Lexid and returns an error if blockSize is 0 or invalid chars
func New(chars string, blockSize, stepSize int) (*Lexid, error) {
	if blockSize < 1 {
		blockSize = 1
	}
	if stepSize < 1 {
		stepSize = 1
	}

	uniqueCharsMap := [256]bool{}
	uniqueChars := make([]byte, 0, len(chars))

	for i := 0; i < len(chars); i++ {
		if !uniqueCharsMap[chars[i]] {
			uniqueCharsMap[chars[i]] = true
			uniqueChars = append(uniqueChars, chars[i])
		}
	}

	if len(uniqueChars) < 2 {
		return nil, errors.New("chars must contain at least two unique characters")
	}

	sort.Slice(uniqueChars, func(i, j int) bool {
		return uniqueChars[i] < uniqueChars[j]
	})

	lower := uniqueChars[0]
	upper := uniqueChars[len(uniqueChars)-1]

	// Initialize the nextChar and charIndex arrays
	nextChar := [256]byte{}
	charIndex := [256]int{}
	for i := range nextChar {
		nextChar[i] = lower
		charIndex[i] = -1
	}
	for i, c := range uniqueChars {
		if i < len(uniqueChars)-1 {
			nextChar[c] = uniqueChars[i+1]
		} else {
			nextChar[c] = uniqueChars[0]
		}
		charIndex[c] = i
	}

	return &Lexid{
		chars:     uniqueChars,
		blockSize: blockSize,
		stepSize:  stepSize,
		lower:     lower,
		upper:     upper,
		nextChar:  nextChar,
		charIndex: charIndex,
	}, nil
}

// Lexid represents a lexicographically sorted ID generator
type Lexid struct {
	chars     []byte
	nextChar  [256]byte
	charIndex [256]int
	blockSize int
	stepSize  int
	lower     byte
	upper     byte
}

// Next generates the next lexicographically sorted string ID
func (l Lexid) Next(prev string) (next string) {
	return l.nextStep(prev, l.stepSize)
}

func (l Lexid) nextStep(prev string, step int) (next string) {
	if prev == "" {
		firstId := make([]byte, l.blockSize)
		for i := range firstId {
			if i == l.blockSize-1 {
				firstId[i] = l.nextChar[l.lower]
			} else {
				firstId[i] = l.lower
			}
		}
		return string(firstId)
	}

	if pad := l.blockSize - (len(prev) % l.blockSize); pad != l.blockSize {
		return l.padding(prev, pad)
	}

	prevBytes := []byte(prev)

	var carry int
	for s := 0; s < step; s++ {
		carry = 1
		for i := len(prevBytes) - 1; i >= 0; i-- {
			if carry == 0 {
				break
			}
			newValue := l.nextChar[prevBytes[i]]
			if newValue == l.lower {
				if i == len(prevBytes)-1 {
					newValue = l.nextChar[l.lower]
				}
				carry = 1
			} else {
				carry = 0
			}
			prevBytes[i] = newValue
		}
		if carry != 0 {
			break
		}
	}
	if carry == 1 {
		return l.padding(prev, l.blockSize)
	}

	return string(prevBytes)
}

func (l Lexid) padding(s string, pad int) string {
	var strBytes = []byte(s)
	for i := 0; i < pad; i++ {
		if i == pad-1 {
			strBytes = append(strBytes, l.nextChar[l.lower])
		} else {
			strBytes = append(strBytes, l.lower)
		}
	}
	return string(strBytes)
}

// NextBefore generates the next lexicographically sorted string ID that is lexicographically less than "before"
func (l Lexid) NextBefore(prev, before string) (string, error) {
	if before <= prev {
		return "", fmt.Errorf("incorrect before value: '%s' less or equal '%s'", before, prev)
	}

	var prevPad, beforePad = prev, before
	// make paddings to be sure we're in blockSize
	if pad := l.blockSize - (len(prevPad) % l.blockSize); pad != l.blockSize {
		prevPad = l.padding(prevPad, pad)
	}
	if pad := l.blockSize - (len(beforePad) % l.blockSize); pad != l.blockSize {
		beforePad = l.padding(beforePad, pad)
	}
	if prev == "" {
		pad := l.blockSize * (len(beforePad) / l.blockSize)
		prevPad = l.padding("", pad)
		if prevPad == beforePad {
			prevPad = l.padding("", pad+l.blockSize)
		}
	}

	lDiff := len(prevPad) - len(beforePad)
	if lDiff > 0 {
		beforePad = l.padding(beforePad, lDiff)
	} else if lDiff < 0 {
		prevPad = l.padding(prevPad, -lDiff)
	}

	dist := l.approxDistance(prevPad, beforePad)
	if dist > 0 {
		step := l.stepSize
		for float64(step)/float64(dist) > 0.3 {
			step = step / 2
		}
		if step > 0 {
			next := l.nextStep(prevPad, step)
			if next < before {
				return next, nil
			}
		}
	}
	next := l.addTail(prevPad)
	if prev > next {
		return "", fmt.Errorf("unable to create id between '%s' and '%s'; result='%s'", prev, before, next)
	}
	return next, nil
}

func (l Lexid) approxDistance(id1, id2 string) (distance int) {
	var size = len(id2)
	if len(id1) < len(id2) {
		size = len(id1)
	}

	multiplier := 1
	for i := size - 1; i >= 0; i-- {
		index1 := l.charIndex[id1[i]]
		index2 := l.charIndex[id2[i]]
		distance += (index2 - index1) * multiplier
		multiplier *= len(l.chars)
	}
	return distance
}

func (l Lexid) addTail(prev string) string {
	middle := len(l.chars) / 2
	prevBytes := []byte(prev)
	prevBytes = append(prevBytes, l.chars[middle])
	return l.padding(string(prevBytes), l.blockSize-1)
}
