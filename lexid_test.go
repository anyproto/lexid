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
		assert.Equal(t, "001", lid.Next(""))
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
		assert.Equal(t, "ZZZ001", lid.Next("ZZZ"))
	})
}

func TestLexid_NextBefore(t *testing.T) {
	t.Run("empty before", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 100)
		_, err := lid.NextBefore("001", "")
		assert.Error(t, err)
	})
	t.Run("empty prev", func(t *testing.T) {
		lid := Must(CharsAlphanumericLower, 3, 10)
		firstString := lid.Next("")
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
