package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartEntry_Withdrawn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		heat int
		want bool
	}{
		{
			name: "reports a withdrawn start entry",
			heat: -1,
			want: true,
		},
		{
			name: "reports a start entry without a heat as not withdrawn",
			heat: 0,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := StartEntry{
				Heat: tt.heat,
			}

			withdrawn := entry.Withdrawn()

			assert.Equal(t, tt.want, withdrawn)
		})
	}
}
