package pack3d

import (
	"math"
	"math/rand"

	"github.com/fogleman/fauxgl"
)

var AxisXRotations []fauxgl.Matrix
var AxisYRotations []fauxgl.Matrix
var AxisZRotations []fauxgl.Matrix

func init() {
	axisDirections := [2]int{-1, 1}
	// This nested loops run 24 times in total - for all possible 90deg rotations and axis directions.
	for i := 0; i < 4; i++ { // every axis 4 times to return to the original position
		for _, s := range axisDirections { // switch axis direction - or +
			for axisId := 1; axisId <= 3; axisId++ { // switch axis (3 axes)
				up := AxisZ.Vector()                                  // z axis
				m := fauxgl.Rotate(up, float64(i)*fauxgl.Radians(90)) // 4x4 rotation matrix around the z axis.
				//fmt.Println(Axis(axisId).Vector().MulScalar(float64(s))) is all axis
				m = m.RotateTo(up, Axis(axisId).Vector().MulScalar(float64(s))) //rotation matrix in all axis(4 by 4)

				if axisId == 1 {
					AxisXRotations = append(AxisXRotations, m) // 8 rotation matrices.
				} else if axisId == 2 {
					AxisYRotations = append(AxisYRotations, m) // 8 rotation matrices.
				} else if axisId == 3 {
					AxisZRotations = append(AxisZRotations, m) // 8 rotation matrices.
				}
			}
		}
	}
}

type Undo struct {
	Index       int
	Rotation    int
	Translation fauxgl.Vector
}

type Item struct {
	Mesh               *fauxgl.Mesh
	Trees              []Tree // struc tree -> []Box, struc Box -> {min, max} vector
	RotationId         int    // index of a rotation within Rotations.
	Translation        fauxgl.Vector
	AvailableRotations []fauxgl.Matrix
}

func (item *Item) Matrix() fauxgl.Matrix {
	return item.AvailableRotations[item.RotationId].Translate(item.Translation)
}

func (item *Item) Copy() *Item {
	dup := *item
	return &dup
}

type Model struct {
	Items     []*Item
	MinVolume float64
	MaxVolume float64
	Deviation float64
}

func NewModel() *Model {
	return &Model{nil, 0, 0, 1}
}

func (m *Model) Add(mesh *fauxgl.Mesh, detail, count int, spacing float64, rotations []fauxgl.Matrix) {
	//spacing is the min required distance between objects
	tree := NewTreeForMesh(mesh, detail, spacing)
	trees := make([]Tree, len(rotations))
	for i, m := range rotations {
		trees[i] = tree.Transform(m)
	}
	for i := 0; i < count; i++ {
		m.add(mesh, trees, rotations)
	}
}

func (m *Model) add(mesh *fauxgl.Mesh, trees []Tree, rotations []fauxgl.Matrix) {
	index := len(m.Items)
	item := Item{mesh, trees, 0, fauxgl.Vector{}, rotations} // the translation is 0 for now
	m.Items = append(m.Items, &item)
	d := 1.0
	for !m.ValidChange(index) {
		item.RotationId = rand.Intn(len(rotations))

		item.Translation = fauxgl.RandomUnitVector().MulScalar(d)
		d *= 1.2
	}
	tree := trees[0]
	m.MinVolume = math.Max(m.MinVolume, tree[0].Volume())
	m.MaxVolume += tree[0].Volume() // what is tree[0]?
}

func (m *Model) Reset() {
	items := m.Items
	m.Items = nil
	m.MinVolume = 0
	m.MaxVolume = 0
	for _, item := range items {
		m.add(item.Mesh, item.Trees, item.AvailableRotations)
	}
}

func (m *Model) Pack(iterations int, callback AnnealCallback, singleStlSize []fauxgl.Vector, frameSize fauxgl.Vector, packItemNum int) (*Model, int) {
	e := 0.5
	runannel, ntime := Anneal(m, 1e0*e, 1e-4*e, iterations, callback, singleStlSize, frameSize, packItemNum)
	annealModel := runannel.(*Model)
	return annealModel, ntime
}

func (m *Model) Meshes() []*fauxgl.Mesh {
	result := make([]*fauxgl.Mesh, len(m.Items))
	for i, item := range m.Items {
		mesh := item.Mesh.Copy()
		mesh.Transform(item.Matrix())
		result[i] = mesh
	}
	return result
}

func (m *Model) Mesh() *fauxgl.Mesh {
	result := fauxgl.NewEmptyMesh()
	for _, mesh := range m.Meshes() {
		result.Add(mesh)
	}
	return result
}

/* This function will return the tranformation matrices of all items */
func (m *Model) Transformation() []fauxgl.Matrix {
	result := make([]fauxgl.Matrix, len(m.Items))
	for i, item := range m.Items {
		result[i] = item.Matrix()
	}
	return result
}

func (m *Model) TreeMeshes() []*fauxgl.Mesh {
	result := make([]*fauxgl.Mesh, len(m.Items))
	for i, item := range m.Items {
		mesh := fauxgl.NewEmptyMesh()
		tree := item.Trees[item.RotationId]
		for _, box := range tree[len(tree)/2:] {
			mesh.Add(fauxgl.NewCubeForBox(box))
		}
		mesh.Transform(fauxgl.Translate(item.Translation))
		result[i] = mesh
	}
	return result
}

func (m *Model) TreeMesh() *fauxgl.Mesh {
	result := fauxgl.NewEmptyMesh()
	for _, mesh := range m.TreeMeshes() {
		result.Add(mesh)
	}
	return result
}

/* This function is to make sure no intersection between objects*/
func (m *Model) ValidChange(i int) bool {
	item1 := m.Items[i]
	tree1 := item1.Trees[item1.RotationId]
	for j := 0; j < len(m.Items); j++ { // go through all other items
		if j == i {
			continue
		}
		item2 := m.Items[j]
		tree2 := item2.Trees[item2.RotationId]

		// Better solution for stoppping "nested" packing,  where one item is packed
		// entirely inside another.
		// The intersection test is surface-based, so if a model has a hollow cavity
		// another model could fit inside without failing the intersection test.
		// We instead now reject full encapsulation of bvm boxes
		b1 := tree1[0].Translate(item1.Translation)
		b2 := tree2[0].Translate(item2.Translation)
		if b1.ContainsBox(b2) || b2.ContainsBox(b1) {
			return false
		}

		if tree1.Intersects(tree2, item1.Translation, item2.Translation) {
			return false
		}
	}
	return true
}

/*True if the passed move it within maximum_packing_area, false in all other cases.*/
func (m *Model) ValidBound(i int, singleStlSize []fauxgl.Vector, frameSize fauxgl.Vector) bool {
	var points []fauxgl.Vector
	var point fauxgl.Vector

	transformation := m.Transformation()[i]
	size := singleStlSize[i]
	halfFrame := frameSize.MulScalar(0.5)
	halfSize := size.MulScalar(0.5)

	// Rotate and then check that the rotation applied to the bound is valid in the for loop.

	// Note: In July 2019, this block of points assignments was updated to translate
	//       the bound to the centerof the box before applying the rotation. See commit 593db93.
	//       That translation was later found at the origin of a nasty regression
	//       which made pack3d generate "nested" packings under certain conditions.
	//       That problematic translation had to be reverted in sc-46802.
	//       Incorrect STL output, prior to the sc-46802 fix, for reference, is in
	//       the folder manual_tests: sc-46802_test. Switch the Meshlab visualisation
	//       from faces to points to be able to spot the nested geometries inside the
	//       neck of the STL bust.
	// Meshes are centred during loading, so bounding corners are +/- halfSize.
	points = append(points, fauxgl.V(-halfSize.X, -halfSize.Y, -halfSize.Z))
	points = append(points, fauxgl.V(halfSize.X, -halfSize.Y, -halfSize.Z))
	points = append(points, fauxgl.V(-halfSize.X, halfSize.Y, -halfSize.Z))
	points = append(points, fauxgl.V(-halfSize.X, -halfSize.Y, halfSize.Z))
	points = append(points, fauxgl.V(halfSize.X, halfSize.Y, -halfSize.Z))
	points = append(points, fauxgl.V(halfSize.X, -halfSize.Y, halfSize.Z))
	points = append(points, fauxgl.V(-halfSize.X, halfSize.Y, halfSize.Z))
	points = append(points, fauxgl.V(halfSize.X, halfSize.Y, halfSize.Z))

	for j := 0; j < 8; j++ {
		point = points[j]
		point = transformation.MulPosition(point)
		point = point.Abs()
		// `frameSize` represents the full build volume dimensions.
		// Packing is centred around the origin, so ensure the absolute coordinate fits
		// within half the frame in each dimension.
		if point.Max(halfFrame) == halfFrame {
			continue
		} else {
			return false
		}
	}

	return true
}

func (m *Model) BoundingBox() fauxgl.Box {
	box := fauxgl.EmptyBox
	for _, item := range m.Items {
		tree := item.Trees[item.RotationId]
		box = box.Extend(tree[0].Translate(item.Translation))
	}
	return box
}

func (m *Model) Volume() float64 {
	return m.BoundingBox().Volume()
}

func (m *Model) Energy() float64 {
	return m.Volume() / m.MaxVolume
}

func (m *Model) DoMove(singleStlSize []fauxgl.Vector, frameSize fauxgl.Vector, packItemNum int) (Undo, int) {

	var i int = 0
	// avoids rand.Intn(0) panic / bug
	if packItemNum > 0 {
		i = rand.Intn(packItemNum)
	}
	item := m.Items[i] // single model
	undo := Undo{i, item.RotationId, item.Translation}
	j := 0
	for {
		j += 1
		if rand.Intn(4) == 0 {
			// rotate, 1/4 of probability
			item.RotationId = rand.Intn(len(item.AvailableRotations)) // do a random rotation, it's a random index
		} else {
			// translate, 3/4 of probability
			offset := Axis(rand.Intn(3) + 1).Vector()                   // Pick a random axis
			offset = offset.MulScalar(rand.NormFloat64() * m.Deviation) // A random translation in x or y or z (vector)
			item.Translation = item.Translation.Add(offset)             // add offset to translation
		}

		if m.ValidChange(i) && m.ValidBound(i, singleStlSize, frameSize) {
			break
		}

		item.RotationId = undo.Rotation
		item.Translation = undo.Translation
		if j >= MAX_MOVE_ATTEMPTS {
			break
		}
	}

	return undo, j
}

func (m *Model) UndoMove(undo Undo) {
	item := m.Items[undo.Index]
	item.RotationId = undo.Rotation
	item.Translation = undo.Translation
}

func (m *Model) Copy() Annealable {
	items := make([]*Item, len(m.Items))
	for i, item := range m.Items {
		items[i] = item.Copy()
	}
	return &Model{items, m.MinVolume, m.MaxVolume, m.Deviation}
}
