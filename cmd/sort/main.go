package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/fogleman/fauxgl"
)

func main() {
	path := os.Args[1]
	_count, _ := strconv.ParseInt(os.Args[2], 0, 0)
	count := int(_count)

	mesh, err := fauxgl.LoadMesh(path)
	if err != nil {
		panic(err)
	}

	mesh.MoveTo(fauxgl.Vector{}, fauxgl.Vector{})

	meshes := make([]*fauxgl.Mesh, 0, count)
	n := len(mesh.Triangles) / count
	for i := 0; i < len(mesh.Triangles); i += n {
		m := fauxgl.NewTriangleMesh(mesh.Triangles[i : i+n])
		meshes = append(meshes, m)
	}

	sort.Slice(meshes, func(i, j int) bool {
		a := meshes[i].BoundingBox().Min
		b := meshes[j].BoundingBox().Min
		a = fauxgl.Vector{X: a.Z, Y: a.X, Z: a.Y}
		b = fauxgl.Vector{X: b.Z, Y: b.X, Z: b.Y}
		return a.Less(b)
	})

	result := fauxgl.NewEmptyMesh()
	for _, mesh := range meshes {
		result.Add(mesh)
		fmt.Println(mesh.BoundingBox())
	}
	result.SaveSTL("out.stl")
}
