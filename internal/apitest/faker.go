package apitest

import (
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
)

// SeedEnv names the environment variable that fixes the faker seed, to repeat a failed run exactly.
const SeedEnv = "APITEST_SEED"

// Faker returns a faker of its own for one test with a random seed, which a failing test logs
// and APITEST_SEED sets to reproduce the data.
func Faker(t testing.TB) *gofakeit.Faker {
	t.Helper()

	seed := rand.Uint64()
	if value := os.Getenv(SeedEnv); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		require.NoError(t, err, "%s must be an unsigned integer", SeedEnv)
		seed = parsed
	}

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("faker seed: %d (rerun with %s=%d)", seed, SeedEnv, seed)
		}
	})

	return gofakeit.New(seed)
}
