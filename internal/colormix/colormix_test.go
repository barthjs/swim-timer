package colormix_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/colormix"
)

func TestWithBlack(t *testing.T) {
	t.Parallel()

	// The expected colors are those Chromium computes for color-mix(in oklch, black 60%, <color>)
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "darkens a gray blue",
			color: "#64748b",
			want:  "#161b23",
		},
		{
			name:  "darkens a pure green",
			color: "#00ff00",
			want:  "#004800",
		},
		{
			name:  "darkens white to gray",
			color: "#ffffff",
			want:  "#484848",
		},
		{
			name:  "accepts three digits",
			color: "#fff",
			want:  "#484848",
		},
		{
			name:  "ignores the alpha of eight digits",
			color: "#f5820d80",
			want:  "#442001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := colormix.WithBlack(tt.color, 60)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects a color that is no hex color", func(t *testing.T) {
		t.Parallel()

		for _, color := range []string{"", "7c3aed", "#7c", "#7c3aed0", "#7c3aeg"} {
			_, err := colormix.WithBlack(color, 60)

			require.ErrorIs(t, err, colormix.ErrInvalidHex, color)
		}
	})
}
