package pack3d_test

import (
	"testing"

	"github.com/Authentise/pack3d/pack3d"
)

func TestConfigDeduping(t *testing.T) {
	input := "../tests/fixtures/duplicated_items.json"
	config, err := pack3d.ParseConfig(input)

	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	if len(config.ConfigItems) != 10 {
		t.Fatalf("Unexpected number of config items prior to deduping")
	}

	config.Dedupe()

	if len(config.ConfigItems) != 1 {
		t.Fatalf("Unexpected number of config items after to deduping: %d", len(config.ConfigItems))
	}

	if config.ConfigItems[0].Count != 10 {
		t.Fatalf("Unexpected first config item count after to deduping: %d", config.ConfigItems[0].Count)
	}
}
