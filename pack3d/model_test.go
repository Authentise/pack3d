package pack3d

import (
	"testing"

	"github.com/fogleman/fauxgl"
)

func TestValidChangeRejectsNestedRootBoxes(t *testing.T) {
	// Two items whose root BVH boxes are fully nested should be rejected.
	// This guards against "nested packings" where surface-only intersection
	// tests allow one model to sit inside a hollow cavity of another.
	bigTree := Tree{
		fauxgl.Box{Min: fauxgl.V(-10, -10, -10), Max: fauxgl.V(10, 10, 10)},
	}
	smallTree := Tree{
		fauxgl.Box{Min: fauxgl.V(-1, -1, -1), Max: fauxgl.V(1, 1, 1)},
	}

	m := &Model{
		Items: []*Item{
			{
				Trees:              []Tree{bigTree},
				RotationId:         0,
				Translation:        fauxgl.Vector{},
				AvailableRotations: []fauxgl.Matrix{fauxgl.Identity()},
			},
			{
				Trees:              []Tree{smallTree},
				RotationId:         0,
				Translation:        fauxgl.Vector{},
				AvailableRotations: []fauxgl.Matrix{fauxgl.Identity()},
			},
		},
	}

	if m.ValidChange(0) {
		t.Fatalf("expected nested root boxes to be rejected")
	}
	if m.ValidChange(1) {
		t.Fatalf("expected nested root boxes to be rejected")
	}
}


