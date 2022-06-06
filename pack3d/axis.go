package pack3d

import "github.com/fogleman/fauxgl"

type Axis uint8

const (
	AxisNone Axis = iota
	AxisX
	AxisY
	AxisZ
)

func (a Axis) Vector() fauxgl.Vector {
	switch a {
	case AxisX:
		return fauxgl.Vector{X: 1, Y: 0, Z: 0}
	case AxisY:
		return fauxgl.Vector{X: 0, Y: 1, Z: 0}
	case AxisZ:
		return fauxgl.Vector{X: 0, Y: 0, Z: 1}
	}
	return fauxgl.Vector{}
}
