package pack3d

import (
	"encoding/json"
	"os"
)

type Config struct {
	BuildVolume [3]float64   `json:"build_volume"`
	Spacing     float64      `json:"spacing"`
	ConfigItems []ConfigItem `json:"items"`
}

func (c *Config) TotalItems() int {
	total := 0
	for _, item := range c.ConfigItems {
		total += item.Count
	}
	return total
}

// Dedupes a config item array by Filename, and increments Count
func (c *Config) Dedupe() {
	// Array of deduped config items
	deduped := make([]ConfigItem, 0)
	// Filenames by their resultant index in the deduped array
	filenames := make(map[string]int)

	for _, item := range c.ConfigItems {
		index, exists := filenames[item.Filename]
		if !exists {
			deduped = append(deduped, item)
			filenames[item.Filename] = len(deduped) - 1
		} else {
			deduped[index].Count += 1
		}
	}
	c.ConfigItems = deduped
}

type ConfigItem struct {
	Filename string    `json:"filename"`
	Scale    float64   `json:"scale"`
	Count    int       `json:"count"`
	Copack   []*Copack `json:"copack,omitempty"`
	AxesLock *AxesLock `json:"axes_lock"`
}

type Copack struct {
	Filename string `json:"filename"`
	// Scale        float64   `json:"scale"`
	// Transformation [4][4]float64 `json:"transformation"`  // ch32838 initially required this field then the requirements changed.
}

// The struct name AxesLock is incorrect and should be replaced
// with MfgOrientation and corrected everywhere else in this file.
// This naming issue was spotted during the handoff to Tyler.
type AxesLock struct {
	ThetaX *float64 `json:"theta_x"`
	ThetaY *float64 `json:"theta_y"`
	ThetaZ *float64 `json:"theta_z"`
}

func ParseConfig(input string) (*Config, error) {
	config := &Config{}
	file, err := os.ReadFile(input)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(file), config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
