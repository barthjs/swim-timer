package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/domain"
)

func TestRecordingStore_List(t *testing.T) {
	t.Parallel()

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewRecordingStore(storage)
		_, err := st.Create(ctx, "valid.jsonl", []byte("x"))
		require.NoError(t, err)
		putFile(t, storage, "recordings/notes.txt", []byte{})
		putFile(t, storage, "recordings/nested/deep.jsonl", []byte("x"))
		putFile(t, storage, "other.jsonl", []byte("x"))

		recordings, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Recording{
			{
				Name: "valid.jsonl",
				Path: "recordings/valid.jsonl",
			},
		}, recordings)
	})
}
