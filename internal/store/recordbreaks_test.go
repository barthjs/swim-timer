package store

import (
	"encoding/json/v2"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestRecordBreakStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the record breaks with the most recent first", func(t *testing.T) {
		t.Parallel()

		st := NewRecordBreakStore(newTestStorage(t))
		older := testRecordBreak(t, 1, 1)
		newer := testRecordBreak(t, 2, 1)
		newer.BrokenAt = older.BrokenAt.Add(time.Minute)
		replaceHeat(t, st, 1, 1, older)
		replaceHeat(t, st, 2, 1, newer)

		breaks, err := st.List(t.Context())

		require.NoError(t, err)
		assert.Equal(t, []domain.RecordBreak{
			newer,
			older,
		}, breaks)
	})

	t.Run("keeps the record breaks in memory after the first read", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)
		_, err := st.List(t.Context())
		require.NoError(t, err)
		putFile(t, storage, recordBreakFile, []byte(`[{"id":"x"}]`))

		breaks, err := st.List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, breaks, "a file changed behind the store is not read again")
	})

	t.Run("returns no record breaks without the file", func(t *testing.T) {
		t.Parallel()

		breaks, err := NewRecordBreakStore(newTestStorage(t)).List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, breaks)
	})
}

func TestRecordBreakStore_LoadAll(t *testing.T) {
	t.Parallel()

	t.Run("reads the file again", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)
		_, err := st.List(t.Context())
		require.NoError(t, err)
		recordBreak := testRecordBreak(t, 1, 1)
		data, err := json.Marshal([]domain.RecordBreak{recordBreak})
		require.NoError(t, err)
		putFile(t, storage, recordBreakFile, data)

		err = st.LoadAll(t.Context())

		require.NoError(t, err)
		breaks, err := st.List(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []domain.RecordBreak{recordBreak}, breaks)
	})

	t.Run("rejects a file that cannot be read and keeps the previous record breaks", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)
		recordBreak := testRecordBreak(t, 1, 1)
		replaceHeat(t, st, 1, 1, recordBreak)
		putFile(t, storage, recordBreakFile, []byte("{"))

		err := st.LoadAll(t.Context())

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "breaks.json", dataErr.File)
		breaks, err := st.List(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []domain.RecordBreak{recordBreak}, breaks)
	})
}

func TestRecordBreakStore_ReplaceHeat(t *testing.T) {
	t.Parallel()

	t.Run("replaces the record breaks of the heat and passes its previous ones and the others", func(t *testing.T) {
		t.Parallel()

		st := NewRecordBreakStore(newTestStorage(t))
		other := testRecordBreak(t, 2, 1)
		replaced := testRecordBreak(t, 1, 1)
		replaceHeat(t, st, 1, 1, replaced, other)
		added := testRecordBreak(t, 1, 1)

		var passedPrevious, passedOthers []domain.RecordBreak
		changed, err := st.ReplaceHeat(t.Context(), 1, 1, func(previous, others []domain.RecordBreak) []domain.RecordBreak {
			passedPrevious, passedOthers = previous, others
			return []domain.RecordBreak{added}
		})

		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, []domain.RecordBreak{replaced}, passedPrevious)
		assert.Equal(t, []domain.RecordBreak{other}, passedOthers)
		breaks, err := st.List(t.Context())
		require.NoError(t, err)
		assert.ElementsMatch(t, []domain.RecordBreak{other, added}, breaks)
	})

	t.Run("reports no change for the record breaks the heat has already", func(t *testing.T) {
		t.Parallel()

		st := NewRecordBreakStore(newTestStorage(t))
		kept := testRecordBreak(t, 1, 1)
		replaceHeat(t, st, 1, 1, kept)

		changed, err := st.ReplaceHeat(t.Context(), 1, 1, func(previous, _ []domain.RecordBreak) []domain.RecordBreak {
			return previous
		})

		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("leaves the file to Flush", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)

		changed, err := st.ReplaceHeat(t.Context(), 1, 1, func(_, _ []domain.RecordBreak) []domain.RecordBreak {
			return []domain.RecordBreak{testRecordBreak(t, 1, 1)}
		})

		require.NoError(t, err)
		assert.True(t, changed)
		assert.NotContains(t, listFiles(t, storage, recordsFolder+"/"), recordBreakFile)
	})
}

func TestRecordBreakStore_Flush(t *testing.T) {
	t.Parallel()

	t.Run("writes the record breaks the heats changed", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)
		recordBreak := testRecordBreak(t, 1, 1)
		_, err := st.ReplaceHeat(t.Context(), 1, 1, func(_, _ []domain.RecordBreak) []domain.RecordBreak {
			return []domain.RecordBreak{recordBreak}
		})
		require.NoError(t, err)

		err = st.Flush(t.Context())

		require.NoError(t, err)
		reloaded := NewRecordBreakStore(storage)
		breaks, err := reloaded.List(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []domain.RecordBreak{recordBreak}, breaks)
	})

	t.Run("writes nothing without a change", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordBreakStore(storage)
		_, err := st.List(t.Context())
		require.NoError(t, err)

		err = st.Flush(t.Context())

		require.NoError(t, err)
		assert.Empty(t, listFiles(t, storage, recordsFolder+"/"))
	})
}

// testRecordBreak returns a record break of a heat with random values.
func testRecordBreak(t *testing.T, event, heat int) domain.RecordBreak {
	t.Helper()

	faker := apitest.Faker(t)
	return domain.RecordBreak{
		ID:           faker.LetterN(16),
		TableID:      faker.LetterN(12),
		TableName:    faker.LetterN(12),
		Abbreviation: faker.LetterN(2),
		Event:        event,
		Heat:         heat,
		Lane:         faker.IntRange(1, 8),
		Record:       testRecord(t),
		Previous:     testRecord(t),
		BrokenAt:     faker.Date().UTC().Truncate(time.Second),
	}
}

// replaceHeat makes the record breaks the ones of a heat and writes them.
func replaceHeat(t *testing.T, st *RecordBreakStore, event, heat int, breaks ...domain.RecordBreak) {
	t.Helper()

	_, err := st.ReplaceHeat(t.Context(), event, heat, func(_, _ []domain.RecordBreak) []domain.RecordBreak {
		return breaks
	})
	require.NoError(t, err)
	require.NoError(t, st.Flush(t.Context()))
}
