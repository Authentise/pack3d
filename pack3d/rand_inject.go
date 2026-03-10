package pack3d

import (
	"math/rand"

	"github.com/fogleman/fauxgl"
)

// testRand, when set, is used instead of the global rand. Tests call
// SetTestRand(rand.New(rand.NewSource(seed))) for deterministic packing.
var testRand *rand.Rand

// SetTestRand sets the RNG used by the packing algorithm. When non-nil, it
// replaces the default (global) RNG. Pass nil to restore default behaviour.
// Used by tests for deterministic, repeatable packing.
func SetTestRand(r *rand.Rand) {
	testRand = r
}

func randIntn(n int) int {
	if testRand != nil {
		return testRand.Intn(n)
	}
	return rand.Intn(n)
}

func randFloat64() float64 {
	if testRand != nil {
		return testRand.Float64()
	}
	return rand.Float64()
}

func randNormFloat64() float64 {
	if testRand != nil {
		return testRand.NormFloat64()
	}
	return rand.NormFloat64()
}

// randInBox returns a random 3D point uniformly distributed within the box
// [min, max] in each dimension. Used for initial placement within build volume.
func randInBox(min, max fauxgl.Vector) fauxgl.Vector {
	return fauxgl.V(
		min.X+randFloat64()*(max.X-min.X),
		min.Y+randFloat64()*(max.Y-min.Y),
		min.Z+randFloat64()*(max.Z-min.Z),
	)
}
