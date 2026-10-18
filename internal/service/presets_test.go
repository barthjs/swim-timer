package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

func TestPresetService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the presets", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		preset := newPreset(t)
		require.NoError(t, svc.Save(ctx, preset.Name, preset))

		names, err := svc.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []string{preset.Name}, names)
	})
}

func TestPresetService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		preset := newPreset(t)
		require.NoError(t, svc.Save(ctx, preset.Name, preset))

		got, err := svc.Get(ctx, preset.Name)

		require.NoError(t, err)
		assert.Equal(t, &preset, got)
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()

		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))

		_, err := svc.Get(t.Context(), apitest.Faker(t).LetterN(10))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()

		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))

		for _, name := range invalidPresetNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := svc.Get(t.Context(), name)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
			})
		}
	})
}

func TestPresetService_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		preset := newPreset(t)

		err := svc.Save(ctx, preset.Name, preset)

		require.NoError(t, err)
		got, err := svc.Get(ctx, preset.Name)
		require.NoError(t, err)
		assert.Equal(t, &preset, got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))

		for _, name := range invalidPresetNames() {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				preset := newPreset(t)
				preset.Name = name

				err := svc.Save(t.Context(), name, preset)

				assertValidation(t, err, "name", "invalid")
				names, err := svc.List(t.Context())
				require.NoError(t, err)
				assert.Empty(t, names)
			})
		}
	})

	t.Run("rejects a preset of another name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		faker := apitest.Faker(t)
		name := faker.LetterN(10)

		err := svc.Save(ctx, name, domain.Preset{
			Name: name + faker.LetterN(5),
		})

		assertValidation(t, err, "name", "invalid")
		_, err = svc.Get(ctx, name)
		assert.ErrorIs(t, err, apperr.ErrNotFound, "a preset with another name is not stored")
	})
}

func TestPresetService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the preset", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		preset := newPreset(t)
		require.NoError(t, svc.Save(ctx, preset.Name, preset))

		err := svc.Delete(ctx, preset.Name)

		require.NoError(t, err)
		_, err = svc.Get(ctx, preset.Name)
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()

		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))

		err := svc.Delete(t.Context(), apitest.Faker(t).LetterN(10))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewPresetService(store.NewPresetStore(newTestStorage(t)))
		preset := newPreset(t)
		require.NoError(t, svc.Save(ctx, preset.Name, preset))

		for _, name := range append(invalidPresetNames(), "x/../"+preset.Name) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := svc.Delete(t.Context(), name)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
				names, err := svc.List(t.Context())
				require.NoError(t, err)
				assert.Equal(t, []string{preset.Name}, names, "a malformed name deletes nothing")
			})
		}
	})
}

// invalidPresetNames returns names that cannot address a preset file.
func invalidPresetNames() []string {
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, strings.Repeat("a", 251)}
}

// newPreset returns a valid preset with a random name and random dimensions.
func newPreset(t *testing.T) domain.Preset {
	t.Helper()

	faker := apitest.Faker(t)
	return domain.Preset{
		Name:        faker.LetterN(10),
		FrameWidth:  faker.IntRange(1, 4000),
		FrameHeight: faker.IntRange(1, 4000),
		TextSize:    faker.IntRange(10, 100),
		LaneGap:     faker.IntRange(0, 100),
		TextAnchor:  domain.TextAnchorLeft,
		Lanes:       map[string]domain.LanePreset{},
	}
}
