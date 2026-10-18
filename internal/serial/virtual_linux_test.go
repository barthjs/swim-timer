package serial

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenVirtual(t *testing.T) {
	t.Parallel()

	t.Run("creates a pseudo-terminal", func(t *testing.T) {
		t.Parallel()

		writer, name, err := OpenVirtual()

		require.NoError(t, err)
		t.Cleanup(func() { _ = writer.Close() })
		assert.Regexp(t, `^/dev/pts/\d+$`, name)
		reader, err := os.OpenFile(name, os.O_RDWR, 0)
		require.NoError(t, err, "the pseudo-terminal is unlocked")
		t.Cleanup(func() { _ = reader.Close() })
		_, err = writer.Write([]byte("frame\n"))
		require.NoError(t, err)
		buf := make([]byte, 5)
		_, err = io.ReadFull(reader, buf)
		require.NoError(t, err)
		assert.Equal(t, "frame", string(buf))
	})
}
