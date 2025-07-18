package lexid

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexid_Next(t *testing.T) {
	t.Run("first id", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		assert.Equal(t, "002", lid.Next(""))
	})
	t.Run("next", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 4, 100)
		var prev, next string

		for i := 0; i < 50000; i++ {
			next = lid.Next(prev)
			assert.Greater(t, next, prev)
			assert.False(t, strings.HasSuffix(next, "0"), next)
			prev = next
		}
	})
	t.Run("padding", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		next := lid.Next("c")
		assert.Equal(t, "c01", next)
		next = lid.Next("c01")
		assert.Equal(t, "c02", next)
	})
	t.Run("next step", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 2)
		assert.Equal(t, "003", lid.Next("001"))
		assert.Equal(t, "005", lid.Next("003"))
		assert.Equal(t, "ZZZ003", lid.Next("ZZZ"))
	})
}

func TestLexid_Middle(t *testing.T) {
	t.Run("alphanumeric lower", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		middle := lid.Middle()
		assert.Len(t, middle, 3)
		assert.Equal(t, "iii", middle)
	})
	t.Run("alphanumeric", func(t *testing.T) {
		lid := Must(CharsAlphanumeric, 4, 1)
		middle := lid.Middle()
		assert.Len(t, middle, 4)
		assert.Equal(t, "VVVV", middle)
	})
	t.Run("base58", func(t *testing.T) {
		lid := Must(CharsBase58, 2, 1)
		middle := lid.Middle()
		assert.Len(t, middle, 2)
		expectedMiddleChar := CharsBase58[len(CharsBase58)/2]
		assert.Equal(t, string([]byte{expectedMiddleChar, expectedMiddleChar}), middle)
	})
	t.Run("zero value returns empty", func(t *testing.T) {
		var lid Lexid // zero value - blockSize=0, so returns empty string
		result := lid.Middle()
		assert.Equal(t, "", result)
	})

	t.Run("empty chars with blockSize panics", func(t *testing.T) {
		lid := Lexid{
			chars:     []byte{}, // empty - invalid state
			blockSize: 3,
		}
		assert.Panics(t, func() {
			lid.Middle()
		})
	})

	t.Run("nil chars panics", func(t *testing.T) {
		lid := Lexid{
			chars:     nil, // nil - invalid state
			blockSize: 3,
		}
		assert.Panics(t, func() {
			lid.Middle()
		})
	})
}

func TestLexid_Prev(t *testing.T) {
	t.Run("simple prev", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		prev := lid.Prev("003")
		assert.Equal(t, "002", prev)
		prev = lid.Prev("002")
		assert.Equal(t, "001", prev)
	})
	t.Run("prev with step", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 2)
		prev := lid.Prev("005")
		assert.Equal(t, "003", prev)
		prev = lid.Prev("003")
		assert.Equal(t, "001", prev)
	})
	t.Run("prev with borrow", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		prev := lid.Prev("b00")
		assert.Equal(t, "azz", prev)
	})
	t.Run("prev adds padding on trailing zero", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		// When result would have trailing zero, add padding with max values
		prev := lid.Prev("001")
		assert.Equal(t, "000zzz", prev)

		// Can continue the sequence
		prev = lid.Prev("000zzz")
		assert.Equal(t, "000zzy", prev)
	})
	t.Run("prev from empty", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		prev := lid.Prev("")
		assert.Equal(t, "zzz", prev)
	})
	t.Run("prev reaches lower bound", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 1)
		// When Prev would create trailing zero, add padding
		prev := lid.Prev("001")
		assert.Equal(t, "000zzz", prev)

		// Can continue going back
		prev = lid.Prev("000zzz")
		assert.Equal(t, "000zzy", prev)
	})

	t.Run("prev with large step overflows bottom", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 5)

		// Going back 5 steps from "005": 005->004->003->002->001->000
		// Since "000" ends with trailing zero, it adds padding
		prev := lid.Prev("005")
		assert.Equal(t, "000zzz", prev)

		// Going back 5 steps from "004": 004->003->002->001->000->add padding->continue 1 step
		// "000" -> "000zzz" -> "000zzy"
		prev = lid.Prev("004")
		assert.Equal(t, "000zzy", prev)

		// Test with longer string - should remove a block when overflowing
		prev = lid.Prev("000005")
		// 000005 going back 5 steps -> 000000, which ends with trailing zero
		// So it adds padding: 000000zzz
		assert.Equal(t, "000000zzz", prev)

		// Test case with longer string and more steps
		prev = lid.Prev("000004")
		// 000004 going back 5 steps -> 000000 -> add padding -> continue 1 step
		assert.Equal(t, "000000zzy", prev)

		// Edge case: step=5 from minimum 3-char ID
		prev = lid.Prev("001")
		// 001 going back 1 step -> 000 -> add padding -> continue 4 steps
		assert.Equal(t, "000zzv", prev)

		// Test with even larger step
		lid10 := Must(CharsAlphanumericLower, 4, 10)
		prev = lid10.Prev("0005")
		// 0005 going back 10 steps: 0005->0004->0003->0002->0001->0000->underflow
		// Since result would be "0000" + negative offset, it underflows and adds padding
		assert.Equal(t, "0000zzzu", prev)
	})
}

func TestLexid_NextBefore(t *testing.T) {
	t.Run("empty before", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		_, err := lid.NextBefore("001", "")
		assert.Error(t, err)
	})
	t.Run("empty prev - min before", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 10)
		firstString := "001"
		nextId, err := lid.NextBefore("", firstString)
		require.NoError(t, err)
		assert.True(t, nextId < firstString)
	})
	t.Run("dyn steps", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		prev := lid.Next("001")
		before := lid.Next(prev)
		for i := 0; i < 10; i++ {
			next, err := lid.NextBefore(prev, before)
			require.NoError(t, err)
			assert.Len(t, next, 3)
			assert.Greater(t, before, next)
			assert.Greater(t, next, prev)
			prev = next
		}
	})
	t.Run("add tail", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		prev := "001"
		before := "002"
		next, err := lid.NextBefore(prev, before)
		require.NoError(t, err)
		assert.Greater(t, before, next)
		assert.Greater(t, next, prev)
	})
	t.Run("short before", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		prev := "001"
		before := "002001"
		next, err := lid.NextBefore(prev, before)
		require.NoError(t, err)
		assert.Greater(t, before, next)
		assert.Greater(t, next, prev)
	})
	t.Run("between min padding", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		prev := "zzz"
		next := "zzz001"
		middle, err := lid.NextBefore(prev, next)
		require.NoError(t, err)
		assert.Greater(t, middle, prev)
		assert.Greater(t, next, middle)
	})
	t.Run("between padding", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		prev := "zzz"
		next := "zzzz01"
		middle, err := lid.NextBefore(prev, next)
		require.NoError(t, err)
		assert.Greater(t, middle, prev)
		assert.Greater(t, next, middle)
		assert.Len(t, middle, 6)
	})
}

func TestPrevSpecificCase(t *testing.T) {
	lid := Must(CharsAlphanumericLower, 4, 1)

	// Test the specific case: Prev("0001") should return "0000zzzz"
	result := lid.Prev("0001")
	assert.Equal(t, "0000zzzz", result)

	// Verify the properties
	assert.True(t, result < "0001", "Result should be less than 0001")
	assert.NotEqual(t, 'a', result[len(result)-1], "Should not end with trailing zero")

	// Test that we can continue calling Prev
	result2 := lid.Prev(result)
	assert.True(t, result2 < result, "Should be able to continue the sequence")

	// Test more cases
	testCases := []string{"0001", "00000001", "b000", "0010"}
	for _, tc := range testCases {
		prev := lid.Prev(tc)
		if prev != "" {
			assert.True(t, prev < tc, "Prev result should be less than input")
		}
	}
}

func TestLargeStepSize(t *testing.T) {
	// Test that stepSize validation works
	// With 2 chars and blockSize=2, max capacity = 2^2 = 4 values
	// So stepSize=4 should be rejected
	t.Run("stepSize too large should error", func(t *testing.T) {
		_, err := New("01", 2, 4)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stepSize (4) must be less than block capacity (4)")
	})

	t.Run("stepSize at limit should work", func(t *testing.T) {
		lid, err := New("01", 2, 3) // 3 < 4, should work
		assert.NoError(t, err)

		first := lid.Next("")
		assert.NotEmpty(t, first)
	})

	t.Run("Must panics on invalid stepSize", func(t *testing.T) {
		assert.Panics(t, func() {
			Must("abc", 2, 9) // 3^2 = 9, so stepSize=9 should panic
		})
	})

	t.Run("large step size with middle value", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 4, 1000)
		current := "hhhh"
		result := lid.Prev(current)

		// With large step size, going back 1000 steps from "hhhh" should work normally
		// since we're far from underflow (capacity is 36^4 = 1,679,616)
		assert.NotEmpty(t, result)
		assert.True(t, result < current, "Prev result should be less than input")
		assert.NotEqual(t, 'a', result[len(result)-1], "Should not end with trailing zero")

		// The result should be same length as input (no underflow)
		assert.Equal(t, len(result), len(current), "Result should be same length as input")

		// Should NOT contain padding blocks for normal case
		assert.NotContains(t, result, "0zzz", "Should not contain padding blocks for normal case")
	})
}

func TestLexid_Fuzzy(t *testing.T) {
	lid := Must(CharsAllNoEscape, 4, 100)
	rand.Seed(time.Now().UnixNano())

	var biggest = 3
	var biggestId string
	printBiggest := func(id string) {
		if len(id) > biggest {
			biggestId = id
			biggest = len(id)
		}
	}

	st := time.Now()

	numIDs := 1000
	ids := make([]string, numIDs)
	ids[0] = lid.Next("")
	for i := 1; i < numIDs; i++ {
		ids[i] = lid.Next(ids[i-1])
		printBiggest(ids[i])
	}
	t.Log("created", len(ids), time.Since(st))

	numInsertions := 1000
	for i := 0; i < numInsertions; i++ {
		pos := rand.Intn(numIDs-1) + 1
		prev := ids[pos-1]
		next := ids[pos]
		st = time.Now()
		// Insert a series of new IDs with random length
		seriesLength := rand.Intn(300) + 1 // Random series length between 1 and 300
		newIDs := make([]string, seriesLength)
		for j := 0; j < seriesLength; j++ {
			newID, err := lid.NextBefore(prev, next)
			if err != nil {
				t.Fatalf("Error generating NextBefore ID: %v", err)
			}
			printBiggest(newID)
			prev = newID
			newIDs[j] = newID
		}

		ids = append(ids[:pos], append(newIDs, ids[pos:]...)...)
		numIDs += seriesLength
		if i%1000 == 0 {
			t.Log("inserted", seriesLength, i, time.Since(st))
		}
	}
	// Verify the list is still sorted
	for i := 1; i < len(ids); i++ {
		assert.Greater(t, ids[i], ids[i-1], "IDs are not sorted")
	}

	t.Log("total:", len(ids), biggestId, biggest)
}

func BenchmarkLexid_Next(b *testing.B) {
	bench := func(b *testing.B, lid *Lexid) {
		b.ReportAllocs()
		var prev, next string
		for i := 0; i < b.N; i++ {
			next = lid.Next(prev)
			prev = next
		}
	}

	b.Run("bs=4;step=1", func(b *testing.B) {
		bench(b, Must(CharsAllNoEscape, 4, 1))
	})
	b.Run("bs=4;step=100", func(b *testing.B) {
		bench(b, Must(CharsAllNoEscape, 4, 100))
	})

}
