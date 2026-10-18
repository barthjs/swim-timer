package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintUpdateCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check updateCheck
		want  []string
	}{
		{
			name: "reports the latest version",
			check: updateCheck{
				Current: "v1.0.0",
				Latest:  "v1.0.0",
			},
			want: []string{"v1.0.0 is the latest version."},
		},
		{
			name: "reports a newer version",
			check: updateCheck{
				Current: "v1.0.0",
				Latest:  "v1.1.0",
				URL:     "https://example.test",
				Newer:   true,
			},
			want: []string{"v1.1.0 is available, v1.0.0 is installed.", "https://example.test", "update apply"},
		},
		{
			name: "reports a newer version that cannot be installed",
			check: updateCheck{
				Current:     "v1.0.0",
				Latest:      "v1.1.0",
				Newer:       true,
				Unsupported: "running in a container",
			},
			want: []string{"cannot update itself: running in a container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			err := printUpdateCheck(&out, tt.check)

			require.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, out.String(), want)
			}
		})
	}
}
