package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestPresetStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the presets sorted by name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewPresetStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "b", testPreset("b")))
		require.NoError(t, st.Save(ctx, "a", testPreset("a")))

		names, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, names)
	})

	t.Run("returns no presets without the folder", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		names, err := st.List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, names)
	})

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		st := NewPresetStore(storage)
		require.NoError(t, st.Save(ctx, "valid", testPreset("valid")))
		writeFile(t, filepath.Join(root, presetsFolder, "notes.txt"), "")
		writeFile(t, filepath.Join(root, presetsFolder, "nested", "deep.json"), "{}")
		writeFile(t, filepath.Join(root, "other.json"), "{}")

		names, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []string{"valid"}, names)
	})
}

func TestPresetStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewPresetStore(newTestStorage(t))
		name := apitest.Faker(t).Name()
		want := testPreset(name)
		require.NoError(t, st.Save(ctx, name, want))

		got, err := st.Get(ctx, name)

		require.NoError(t, err)
		assert.Equal(t, want, *got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		for _, name := range invalidFileNames(presetExtension) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := st.Get(t.Context(), name)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		_, err := st.Get(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeFile(t, filepath.Join(root, presetsFolder, "broken.json"), "{not json")

		_, err := NewPresetStore(storage).Get(t.Context(), "broken")

		require.Error(t, err)
		assert.NotErrorIs(t, err, apperr.ErrNotFound, "a broken file is a decoding error, not a missing preset")
	})
}

// writeFile writes content to path and creates the missing parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestPresetStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewPresetStore(newTestStorage(t))
		name := apitest.Faker(t).Name()
		want := testPreset(name)

		err := st.Save(ctx, name, want)

		require.NoError(t, err)
		got, err := st.Get(ctx, name)
		require.NoError(t, err)
		assert.Equal(t, want, *got)
	})

	t.Run("replaces the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewPresetStore(newTestStorage(t))
		require.NoError(t, st.Save(ctx, "a", testPreset("a")))
		second := testPreset("a")
		second.TextSize = 80
		delete(second.Lanes, "2")

		err := st.Save(ctx, "a", second)

		require.NoError(t, err)
		got, err := st.Get(ctx, "a")
		require.NoError(t, err)
		assert.Equal(t, second, *got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		for _, name := range invalidFileNames(presetExtension) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := st.Save(t.Context(), name, testPreset(name))

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})
}

func TestPresetStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewPresetStore(newTestStorage(t))
		name := apitest.Faker(t).Name()
		require.NoError(t, st.Save(ctx, name, testPreset(name)))

		err := st.Delete(ctx, name)

		require.NoError(t, err)
		_, err = st.Get(ctx, name)
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		for _, name := range invalidFileNames(presetExtension) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := st.Delete(t.Context(), name)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()

		st := NewPresetStore(newTestStorage(t))

		err := st.Delete(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

// testPreset returns a preset with two lanes under the given name.
func testPreset(name string) domain.Preset {
	comment := "camera on the left"
	return domain.Preset{
		Name:         name,
		Comment:      &comment,
		FrameWidth:   1920,
		FrameHeight:  1080,
		TextSize:     64,
		LaneGap:      19,
		ReverseLanes: true,
		TextAnchor:   domain.TextAnchorRight,
		Lanes: map[string]domain.LanePreset{
			"1": {
				TL: domain.Point{
					X: 0.01,
					Y: 0.02,
				},
				TR: domain.Point{
					X: 0.9,
					Y: 0.025,
				},
				BR: domain.Point{
					X: 0.95,
					Y: 0.08,
				},
				BL: domain.Point{
					X: 0.005,
					Y: 0.07,
				},
			},
			"2": {
				TL: domain.Point{
					X: 0.005,
					Y: 0.09,
				},
				TR: domain.Point{
					X: 0.95,
					Y: 0.095,
				},
				BR: domain.Point{
					X: 0.99,
					Y: 0.16,
				},
				BL: domain.Point{
					X: 0.001,
					Y: 0.15,
				},
				Locked: true,
			},
		},
	}
}
