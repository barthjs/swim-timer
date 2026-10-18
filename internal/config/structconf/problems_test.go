package structconf

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblems_Err(t *testing.T) {
	t.Parallel()

	t.Run("collects every error on its own line", func(t *testing.T) {
		t.Parallel()

		first := errors.New("a.first is required")
		var p Problems
		p.Add(first)
		p.Addf("a.second must be at most %d", 3)
		p.Add(errors.Join(errors.New("b.third is invalid"), errors.New("b.fourth is invalid")))

		err := p.Err()

		require.Error(t, err)
		assert.ErrorIs(t, err, first)
		assert.EqualError(t, err, "a.first is required\na.second must be at most 3\nb.third is invalid\nb.fourth is invalid")
	})

	t.Run("keeps a wrapped joined error as one problem", func(t *testing.T) {
		t.Parallel()

		var p Problems
		p.Add(fmt.Errorf("a.list is invalid: %w", errors.Join(errors.New("first"), errors.New("second"))))

		err := p.Err()

		assert.EqualError(t, err, "a.list is invalid: first\nsecond")
	})

	t.Run("returns nil without problems", func(t *testing.T) {
		t.Parallel()

		var p Problems
		p.Add(nil)

		err := p.Err()

		assert.NoError(t, err)
	})
}
