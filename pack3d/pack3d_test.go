package pack3d_test

import (
	"testing"

	"github.com/Authentise/pack3d/pack3d"
)

func testPackingInputFile(t *testing.T, input string) {
	config, err := pack3d.ParseConfig(input)

	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	output, err := pack3d.Pack(config)

	if err != nil {
		t.Fatalf("Failed to pack config: %s", err)
	}

	if len(output.Model.Items) != config.TotalItems() {
		t.Fatalf("Failed to pack all models: packed %d of %d", len(output.Model.Items), config.TotalItems())
	}
}

func TestCoPack(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/copack.json")
}

// Tests an input file with old master-ricoh api
// These input files don't have axes_lock and spacing fields
func TestMasterRicohCoPack(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/master-ricoh-copack.json")
}

func TestBuildPlateTooSmall(t *testing.T) {
	// This test has been removed since it takes so long.
	// Pack3d's algorithm attempts to pack using binary search to find
	// the optimal number of items. Each step takes at least 10s, and
	// so failing packs take a long time to run.
	t.Skip()

	testPackingInputFile(t, "../tests/fixtures/too-small.json")
}

func TestCh32838(t *testing.T) {

	testPackingInputFile(t, "../tests/fixtures/ch32838.json")
}

func TestSc46802(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc46802.json")
}
