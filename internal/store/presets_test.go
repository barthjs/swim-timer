package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeFile(t, filepath.Join(root, presetsFolder, "broken.json"), "{not json")

		_, err := NewPresetStore(storage).Get(t.Context(), "broken")

		require.Error(t, err)
		assert.NotErrorIs(t, err, apperr.ErrNotFound, "a broken file is a decoding error, not a missing preset")
	})
}

// testPreset returns a preset with two lanes under the given name.
func testPreset(name string) domain.Preset {
	description := "camera on the left"
	return domain.Preset{
		Name:         name,
		Description:  &description,
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
