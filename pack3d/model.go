package pack3d

import (
	"math"
	"math/rand"

	"github.com/fogleman/fauxgl"
)

var Rotations []fauxgl.Matrix

/*The loop runs 24 times for all the rotation possibility*/
func init() {
	axisDirections := [2]int{-1, 1}
	for i := 0; i < 4; i++ { // every axis 4 times to return to the original position
		for _, s := range axisDirections{  // switch axis direction - or +
			for a := 1; a <= 3; a++ { // switch axis (3 axis)
				up := AxisZ.Vector() // z axis
				m := fauxgl.Rotate(up, float64(i)*fauxgl.Radians(90)) // Rotation matrix in z axis (4 by 4 matrix)
				//fmt.Println(Axis(a).Vector().MulScalar(float64(s))) is all axis
				m = m.RotateTo(up, Axis(a).Vector().MulScalar(float64(s))) //rotation matrix in all axis(4 by 4)
				Rotations = append(Rotations, m) // 24 rotation matrices
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
	Mesh        *fauxgl.Mesh
	Trees       []Tree // struc tree -> []Box, struc Box -> {min, max} vector
	Rotation    int
	Translation fauxgl.Vector
}

func (item *Item) Matrix() fauxgl.Matrix {
	return Rotations[item.Rotation].Translate(item.Translation)
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

func (m *Model) Add(mesh *fauxgl.Mesh, detail, count int, spacing float64){
	//spacing is the min required distance between objects
	tree := NewTreeForMesh(mesh, detail, spacing)
	trees := make([]Tree, len(Rotations))
	for i, m := range Rotations {
		trees[i] = tree.Transform(m)
	}
	for i := 0; i < count; i++ {
		m.add(mesh, trees)
	}
}

func (m *Model) add(mesh *fauxgl.Mesh, trees []Tree) {
	index := len(m.Items)
	item := Item{mesh, trees, 0, fauxgl.Vector{}} // the translation is 0 for now
	m.Items = append(m.Items, &item)
	d := 1.0
	for !m.ValidChange(index) {
		item.Rotation = rand.Intn(len(Rotations))

		item.Translation = fauxgl.RandomUnitVector().MulScalar(d)
		d *= 1.2
	}
	tree := trees[0]
	m.MinVolume = math.Max(m.MinVolume, tree[0].Volume())
	m.MaxVolume += tree[0].Volume()  // what is tree[0]?
}

func (m *Model) Reset() {
	items := m.Items
	m.Items = nil
	m.MinVolume = 0
	m.MaxVolume = 0
	for _, item := range items {
		m.add(item.Mesh, item.Trees)
	}
}

func (m *Model) Pack(iterations int, callback AnnealCallback, singleStlSize []fauxgl.Vector, frameSize fauxgl.Vector, packItemNum int) (*Model, int) {
	e := 0.5
	runannel, ntime:= Anneal(m, 1e0*e, 1e-4*e, iterations, callback, singleStlSize, frameSize, packItemNum)
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
		tree := item.Trees[item.Rotation]
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
	tree1 := item1.Trees[item1.Rotation]
	for j := 0; j < len(m.Items); j++ {  // go through all other items
		if j == i {
			continue
		}
		item2 := m.Items[j]
		tree2 := item2.Trees[item2.Rotation]
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

	// Rotate around the center of volume while checking if rotation is valid. (do not rotate around origin)
	points = append(points,fauxgl.V(size.X/2, size.Y/2, size.Z/2))
	points = append(points,fauxgl.V(size.X/2, -size.Y/2, size.Z/2))
	points = append(points,fauxgl.V(size.X/2, -size.Y/2, -size.Z/2))
	points = append(points,fauxgl.V(size.X/2, size.Y/2, -size.Z/2))
	points = append(points,fauxgl.V(-size.X/2, size.Y/2, size.Z/2))
	points = append(points,fauxgl.V(-size.X/2, size.Y/2, -size.Z/2))
	points = append(points,fauxgl.V(-size.X/2, -size.Y/2, size.Z/2))
	points = append(points,fauxgl.V(-size.X/2, -size.Y/2, -size.Z/2))

	for j:=0; j<8; j++{
		point = points[j]
		point = transformation.MulPosition(point)
		point = point.Abs()
		if point.Max(frameSize) == frameSize {
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
		tree := item.Trees[item.Rotation]
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
	i := rand.Intn(packItemNum) // choose a random index in models
	item := m.Items[i]  // single model
	undo := Undo{i, item.Rotation, item.Translation}
	j := 0
	for {
		j += 1
		if rand.Intn(4) == 0 {
			// rotate, 1/4 of probability
			item.Rotation = rand.Intn(len(Rotations)) // do a random rotation, it's a random index
		} else {
			// translate, 3/4 of probability
			offset := Axis(rand.Intn(3) + 1).Vector()  // Pick a random axis
			offset = offset.MulScalar(rand.NormFloat64() * m.Deviation)  // A random translation in x or y or z (vector)
			item.Translation = item.Translation.Add(offset)  // add offset to translation
		}

		if m.ValidChange(i) && m.ValidBound(i, singleStlSize, frameSize) {
			break
		}

		item.Rotation = undo.Rotation
		item.Translation = undo.Translation
		if j>=100 {
			break
		}
	}


	return undo, j
}

func (m *Model) UndoMove(undo Undo) {
	item := m.Items[undo.Index]
	item.Rotation = undo.Rotation
	item.Translation = undo.Translation
}

func (m *Model) Copy() Annealable {
	items := make([]*Item, len(m.Items))
	for i, item := range m.Items {
		items[i] = item.Copy()
	}
	return &Model{items, m.MinVolume, m.MaxVolume, m.Deviation}
}

