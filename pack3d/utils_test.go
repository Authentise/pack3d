package pack3d_test

import (
	"testing"

	"github.com/Authentise/pack3d/pack3d"
	"github.com/fogleman/fauxgl"
)

func TestAvailableRotations(t *testing.T) {
	config := pack3d.ConfigItem{
		AxesLock: &pack3d.AxesLock{
			ThetaX: nil,
			ThetaY: nil,
			ThetaZ: nil,
		},
	}

	rot := config.AvailableRotations()

	if len(rot) != 24 {
		t.Fatal("Unexpected number of rotations for unlocked axes")
	}
	zero := 0.0
	config.AxesLock.ThetaX = &zero

	rot = config.AvailableRotations()

	if len(rot) != 16 {
		t.Fatal("Unexpected number of rotations for locked X axis")
	}

	config.AxesLock.ThetaY = &zero

	rot = config.AvailableRotations()

	if len(rot) != 8 {
		t.Fatal("Unexpected number of rotations for locked X and Y axes")
	}

	config.AxesLock.ThetaZ = &zero

	rot = config.AvailableRotations()

	// Identity
	if len(rot) != 1 {
		t.Fatal("Unexpected number of rotations for locked X, Y, Z axes")
	}

	// Legacy master-ricoh case, axes lock was not supplied
	config.AxesLock = nil
	rot = config.AvailableRotations()

	// All unlocked
	if len(rot) != 24 {
		t.Fatal("Unexpected number of rotations for nil axes lock")
	}
}

func TestManufacturingOrientation(t *testing.T) {
	config := pack3d.ConfigItem{
		AxesLock: &pack3d.AxesLock{
			ThetaX: nil,
			ThetaY: nil,
			ThetaZ: nil,
		},
	}

	mtx := config.ManufacturingOrientation()


	if mtx != fauxgl.Identity() {
		t.Fatal("Unexpected manufacturing transformation matrix for unlocked axes")
	}

	// Legacy master-ricoh case, axes lock was not supplied
	config.AxesLock = nil
	mtx = config.ManufacturingOrientation()
	if mtx != fauxgl.Identity() {
		t.Fatal("Unexpected manufacturing transformation matrix for nil axes lock")
	}
}
