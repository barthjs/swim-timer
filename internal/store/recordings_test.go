package store

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestRecordingStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the recordings sorted by name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		f := apitest.Faker(t)
		st := NewRecordingStore(newTestStorage(t))
		_, err := st.Create(ctx, "zeta.jsonl", []byte(f.Sentence(6)))
		require.NoError(t, err)
		_, err = st.Create(ctx, "alpha.jsonl", []byte(f.Sentence(6)))
		require.NoError(t, err)

		recordings, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Recording{
			{
				Name: "alpha.jsonl",
				Path: "recordings/alpha.jsonl",
			},
			{
				Name: "zeta.jsonl",
				Path: "recordings/zeta.jsonl",
			},
		}, recordings)
	})

	t.Run("returns no recordings without the folder", func(t *testing.T) {
		t.Parallel()

		recordings, err := NewRecordingStore(newTestStorage(t)).List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, recordings)
	})

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

func TestRecordingStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the recording", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewRecordingStore(newTestStorage(t))
		content := []byte(apitest.Faker(t).Sentence(6))
		recording, err := st.Create(ctx, "alpha.jsonl", content)
		require.NoError(t, err)

		reader, err := st.Get(ctx, recording.Path)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for an unknown recording", func(t *testing.T) {
		t.Parallel()

		_, err := NewRecordingStore(newTestStorage(t)).Get(t.Context(), "recordings/missing.jsonl")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordingStore_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the recording", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		content := []byte(apitest.Faker(t).Sentence(6))

		recording, err := NewRecordingStore(storage).Create(t.Context(), "alpha.jsonl", content)

		require.NoError(t, err)
		assert.Equal(t, domain.Recording{
			Name: "alpha.jsonl",
			Path: "recordings/alpha.jsonl",
		}, recording)
		assert.Equal(t, content, readFile(t, storage, "recordings/alpha.jsonl"))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordingStore(storage)

		for _, name := range invalidRecordingNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := st.Create(t.Context(), name, []byte("x"))

				assert.ErrorIs(t, err, ErrInvalidName)
				assert.Empty(t, listFiles(t, storage, ""))
			})
		}
	})
}

func TestRecordingStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the recording", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewRecordingStore(storage)
		_, err := st.Create(ctx, "alpha.jsonl", []byte("x"))
		require.NoError(t, err)

		err = st.Delete(ctx, "alpha.jsonl")

		require.NoError(t, err)
		assert.Empty(t, listFiles(t, storage, "recordings/"))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordingStore(storage)
		_, err := st.Create(t.Context(), "alpha.jsonl", []byte("x"))
		require.NoError(t, err)

		for _, name := range invalidRecordingNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := st.Delete(t.Context(), name)

				assert.ErrorIs(t, err, ErrInvalidName)
				assert.Equal(t, []string{"recordings/alpha.jsonl"}, listFiles(t, storage, "recordings/"))
			})
		}
	})

	t.Run("returns not found for an unknown recording", func(t *testing.T) {
		t.Parallel()

		err := NewRecordingStore(newTestStorage(t)).Delete(t.Context(), "missing.jsonl")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordingStore_Path(t *testing.T) {
	t.Parallel()

	t.Run("returns the file of the recording", func(t *testing.T) {
		t.Parallel()

		p, err := NewRecordingStore(newTestStorage(t)).Path("alpha.jsonl")

		require.NoError(t, err)
		assert.Equal(t, "recordings/alpha.jsonl", p)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewRecordingStore(newTestStorage(t))

		for _, name := range invalidRecordingNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := st.Path(name)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})
}

// invalidRecordingNames returns names that cannot address an recording file,
// the invalid file names with the recording extension and names without it.
func invalidRecordingNames() []string {
	names := []string{"", "alpha", "alpha.json"}
	for _, name := range invalidFileNames(recordingExtension) {
		names = append(names, name+recordingExtension)
	}
	return names
}
