package pack3d

import "math/rand"

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
