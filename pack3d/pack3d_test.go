package pack3d_test

import (
	"fmt"
	"testing"

	"github.com/Authentise/pack3d/pack3d"
)

func TestCoPack(t *testing.T) {
	config, err := pack3d.ParseConfig("../tests/fixtures/copack.json")

	fmt.Printf("%v", config)
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	_, err = pack3d.Pack(config)

	if err != nil {
		t.Fatalf("Failed to pack model: %s", err)
	}
}

// Tests an input file with old master-ricoh api
// These input files don't have axes_lock and spacing fields
func TestMasterRicohCoPack(t *testing.T) {
	config, err := pack3d.ParseConfig("../tests/fixtures/master-ricoh-copack.json")

	fmt.Printf("%v", config)
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	_, err = pack3d.Pack(config)

	if err != nil {
		t.Fatalf("Failed to pack model: %s", err)
	}
}
