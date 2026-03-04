package pack3d_test

import (
	"encoding/json"
	"math/rand"
	"os"
	"testing"

	"github.com/Authentise/pack3d/pack3d"
)

func isZeroTransformation(tform [4][4]float64) bool {
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if tform[r][c] != 0 {
				return false
			}
		}
	}
	return true
}

func countPacked(transMaps []pack3d.TransMap) int {
	packed := 0
	for _, tm := range transMaps {
		if !isZeroTransformation(tm.Transformation) {
			packed++
		}
	}
	return packed
}

func testPackingInputFile(t *testing.T, input string, expectedPacked int) {
	// The packing algorithm uses global randomness (math/rand). Seed it so that
	// "expectedPacked" assertions are repeatable across runs.
	//
	// Note: Tests are not marked t.Parallel, so a global seed is safe here.
	rand.Seed(1)

	config, err := pack3d.ParseConfig(input)

	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	output, err := pack3d.Pack(config)

	if err != nil {
		t.Fatalf("Failed to pack config: %s", err)
	}

	var transMaps []pack3d.TransMap
	if err := json.Unmarshal(output.MeshJSON, &transMaps); err != nil {
		t.Fatalf("Failed to decode packing output JSON: %s", err)
	}

	// Assert output shape is consistent with how many packable items were provided.
	// This remains stable even if only a subset can be packed into the build volume.
	if len(transMaps) != config.TotalItems() {
		t.Fatalf("Unexpected number of output items: got %d, want %d", len(transMaps), config.TotalItems())
	}

	gotPacked := countPacked(transMaps)
	if gotPacked != expectedPacked {
		t.Fatalf("Unexpected number packed: got %d, want %d", gotPacked, expectedPacked)
	}
}

func TestCoPack(t *testing.T) {
	// Expected packed count is recorded as a regression target.
	// If this changes, it may indicate a behavioural change in packing heuristics.
	testPackingInputFile(t, "../tests/fixtures/copack.json", 17)
}

// Tests an input file with old master-ricoh api
// These input files don't have axes_lock and spacing fields
func TestMasterRicohCoPack(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/master-ricoh-copack.json", 7)
}

func TestBuildPlateTooSmall(t *testing.T) {
	// This fixture is intentionally too small.
	testPackingInputFile(t, "../tests/fixtures/too-small.json", 0)
}

func TestPartialPack(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/partial-pack.json", 1)
}

func TestSc45665(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc45665.json", 10)
}

func TestCh32838(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow fixture in -short mode")
	}
	// Overflow fixtures is sensitive to behavioural changes.
	// With copack items treated as independently-packable and locked seed
	// the packed count should remain stable at 79
	testPackingInputFile(t, "../tests/fixtures/ch32838.json", 79)
}

func TestSc46802(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc46802.json", 4)
}

func TestSc44515(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc44515.json", 17)
}

// There was a rand.Intn(0) crash that the attached benchy regularly triggers. A test to make sure that happens and completes
func TestZeroIndexCrashFixed(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/input_benchy_zero_crash.json", 10)
}

// There was a rand.Intn(0) crash that the attached benchy regularly triggers.
// Run that packing for many minutes, until failure, there is not way to fit that volume of prints into that build space
func TestLogoCubeCorner(t *testing.T) {
	// Pack logo, cube, and corner into 7.5x7.5x5 build volume with spacing 2.
	testPackingInputFile(t, "../tests/fixtures/logo_cube_corner.json", 2)
}

func TestOverpackPackEnds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running overpack regression test in -short mode")
	}
	if os.Getenv("PACK3D_LONG_TESTS") != "1" {
		t.Skip("Skipping long-running overpack regression test (set PACK3D_LONG_TESTS=1 to enable)")
	}
	testPackingInputFile(t, "../tests/fixtures/overpack_ends_w_success.json", 5) // TODO: measure and lock in
}
