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

// Pass in the expected number of items packed
func testPartialPackingInputFile(t *testing.T, input string, expectedPacking int) {
	config, err := pack3d.ParseConfig(input)

	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	output, err := pack3d.Pack(config)

	if err != nil {
		t.Fatalf("Failed to pack config: %s", err)
	}

	if len(output.Model.Items) != expectedPacking {
		t.Fatalf("Failed to pack all models: packed %d of %d", len(output.Model.Items), expectedPacking)
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
	// NOTE: This test fails in master-ricoh but NOT in dev, and doesn't fail here.
	t.Skip()
	testPartialPackingInputFile(t, "../tests/fixtures/too-small.json", 1)
}

func TestPartialPack(t *testing.T) {
	// NOTE: This test fails in master-ricoh but NOT in dev, and doesn't fail here.
	t.Skip()
	testPartialPackingInputFile(t, "../tests/fixtures/partial-pack.json", 2) // Currently 3
}

func TestSc45665(t *testing.T) {
	// NOTE: This test fails in master-ricoh but NOT in dev, and doesn't fail here.
	t.Skip()
	testPartialPackingInputFile(t, "../tests/fixtures/sc45665.json", 9) // Currently 10
}

func TestCh32838(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/ch32838.json")
}

func TestSc46802(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc46802.json")
}

func TestSc44515(t *testing.T) {
	testPackingInputFile(t, "../tests/fixtures/sc44515.json")
}
