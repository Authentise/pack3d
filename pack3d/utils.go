package pack3d

import (
	"fmt"
	"time"

	"github.com/fogleman/fauxgl"
)

/* This function returns the current time (it is a timer). */
func timed(name string) func() {
	if len(name) > 0 {
		fmt.Printf("%s... ", name)
	}
	start := time.Now()
	return func() {
		fmt.Println(time.Since(start))
	}
}

// This function returns only the available rotations
// which depend on the unlocked axes provided by the user.
// An unlocked axis is characterised by a `nil` theta angle.
// Setting a theta angle with a Float instead means that
// that rotation axis is locked to a specific angle.
func (c *ConfigItem) AvailableRotations() []fauxgl.Matrix {
	availableRotations := make([]fauxgl.Matrix, 0)

	if c.AxesLock == nil || c.AxesLock.ThetaX == nil {
		availableRotations = append(availableRotations, AxisXRotations...)
	}
	if c.AxesLock == nil || c.AxesLock.ThetaY == nil {
		availableRotations = append(availableRotations, AxisYRotations...)
	}
	if c.AxesLock == nil || c.AxesLock.ThetaZ == nil {
		availableRotations = append(availableRotations, AxisZRotations...)
	}
	// the function needs to return at least one dummy rotation (the identity matrix).
	if len(availableRotations) == 0 {
		availableRotations = append(availableRotations, fauxgl.Identity())
	}
	return availableRotations
}

// This function's purpose is to create a composite rotation matrix from the three provided angles.

// Tech debt: this function might need to be moved into a function in fauxgl.mesh.

// NOTE: The THREE.Euler's rotation order (in Rapidfab) has been set as 'ZYX' to match Blender's rotation order
//
//	and pack3d "seems" to be the same order of rotation but with the "minus" sign for all three angles.
//	e.g.: -fauxgl.Radians(*item.AxesLock.ThetaX)
func (c *ConfigItem) ManufacturingOrientation() fauxgl.Matrix {
	mfgRotationMtx := fauxgl.Identity()
	if c.AxesLock == nil {
		return mfgRotationMtx
	}
	if c.AxesLock.ThetaX != nil {
		axisX := AxisX.Vector() // x axis
		mfgRotationMtx = mfgRotationMtx.Rotate(axisX, -fauxgl.Radians(*c.AxesLock.ThetaX))
	}
	if c.AxesLock.ThetaY != nil {
		axisY := AxisY.Vector() // y axis
		mfgRotationMtx = mfgRotationMtx.Rotate(axisY, -fauxgl.Radians(*c.AxesLock.ThetaY))
	}
	if c.AxesLock.ThetaZ != nil {
		axisZ := AxisZ.Vector() // z axis
		mfgRotationMtx = mfgRotationMtx.Rotate(axisZ, -fauxgl.Radians(*c.AxesLock.ThetaZ))
	}
	return mfgRotationMtx
}
