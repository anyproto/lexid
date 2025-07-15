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
		t.Log(next)
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
		t.Log(next)
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
		t.Log(middle)
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
	t.Logf("Prev(\"0001\") = %q", result)
	assert.Equal(t, "0000zzzz", result)

	// Verify the properties
	assert.True(t, result < "0001", "Result should be less than 0001")
	assert.NotEqual(t, 'a', result[len(result)-1], "Should not end with trailing zero")

	// Test that we can continue calling Prev
	result2 := lid.Prev(result)
	t.Logf("Prev(%q) = %q", result, result2)
	assert.True(t, result2 < result, "Should be able to continue the sequence")

	// Test more cases
	t.Log("\nTesting more cases:")
	testCases := []string{"0001", "00000001", "b000", "0010"}
	for _, tc := range testCases {
		prev := lid.Prev(tc)
		t.Logf("Prev(%q) = %q", tc, prev)
		if prev != "" {
			assert.True(t, prev < tc, "Prev result should be less than input")
		}
	}
}

func TestLexid_Fuzzy(t *testing.T) {
	lid := Must(CharsAllNoEscape, 4, 100)
	rand.Seed(time.Now().UnixNano())

	var biggest = 3
	var biggestId string
	printBiggest := func(id string) {
		if len(id) > biggest {
			t.Log(id)
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
