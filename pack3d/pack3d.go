package pack3d

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/fogleman/fauxgl"
)

const (
	BVH_DETAIL           = 8
	ANNEALING_ITERATIONS = 2000000 // # of trials
)

var NULL_TRANSFORMATION = fauxgl.Matrix{
	X00: 0, X01: 0, X02: 0, X03: 0,
	X10: 0, X11: 0, X12: 0, X13: 0,
	X20: 0, X21: 0, X22: 0, X23: 0,
	X30: 0, X31: 0, X32: 0, X33: 0,
}

type PackingOutput struct {
	Model    *Model
	MeshJSON []byte
}

type TransMap struct {
	Filename          string
	Transformation    [4][4]float64
	VolumeWithSpacing float64
}

// Object to pack
type Object struct {
	filename       string
	scale          fauxgl.Matrix
	mfgRotation    fauxgl.Matrix
	transformation TransMap
}

type Packer struct {
	objects []Object

	model  *Model
	volume float64
	config *Config
	sizes  []fauxgl.Vector
}

func NewPacker(config *Config) (Packer, error) {
	packer := Packer{}
	err := packer.loadConfig(config)
	return packer, err
}

// loadedItem holds a single mesh ready to add to the model, used for sorting by size.
type loadedItem struct {
	object   Object
	size     fauxgl.Vector
	mesh     *fauxgl.Mesh
	count    int
	volume   float64
	spacing  float64
	rotations []fauxgl.Matrix
}

func (p *Packer) loadConfig(config *Config) error {
	// Initialize packer fields
	p.config = config
	p.model = NewModel()
	p.objects = make([]Object, 0, len(config.ConfigItems))
	p.sizes = make([]fauxgl.Vector, 0, len(config.ConfigItems))

	var loaded []loadedItem

	for _, item := range config.ConfigItems {
		filenames := []string{item.Filename}
		for _, cp := range item.Copack {
			filenames = append(filenames, cp.Filename)
		}

		for _, filename := range filenames {
			object := Object{}

			// 1. load the mesh.
			done := timed(fmt.Sprintf("loading mesh %s", filename))
			mesh, err := fauxgl.LoadMesh(filename)
			if err != nil {
				return err
			}
			done()

			// 2. mesh centring.
			mesh.Center()

			// 3. apply scaling (before bounding box / volume).
			object.scale = fauxgl.Scale(fauxgl.V(item.Scale, item.Scale, item.Scale))
			if item.Scale != 1.0 {
				done = timed("scaling mesh")
				mesh.Transform(object.scale)
				done()
			}

			// 4. apply manufacturing rotation (before bounding box / volume).
			// IMPORTANT: do not confuse manufacturing orientation with the packing
			// orientations from the annealing further on.
			object.mfgRotation = item.ManufacturingOrientation()
			mesh.Transform(object.mfgRotation)

			// 5. sizes / bookkeeping for output.
			size := mesh.BoundingBox().Size()
			object.filename = filename

			vol := mesh.BoundingBox().Volume() * float64(item.Count)
			loaded = append(loaded, loadedItem{
				object:    object,
				size:      size,
				mesh:      mesh,
				count:     item.Count,
				volume:    vol,
				spacing:   config.Spacing / 2,
				rotations: item.AvailableRotations(),
			})

			fmt.Printf("  %d triangles\n", len(mesh.Triangles))
			fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)
			fmt.Println("______________________________________________________")
		}
	}

	// Sort by volume (smallest first) so that when we try "pack N", we pack the
	// N smallest items. This ensures we pack at least the corner when the logo
	// and cube are too large for the build volume.
	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].volume < loaded[j].volume
	})

	for _, li := range loaded {
		for i := 0; i < li.count; i++ {
			p.objects = append(p.objects, li.object)
			p.sizes = append(p.sizes, li.size)
		}
		p.volume += li.volume
		done := timed("building bvh tree")
		p.model.Add(li.mesh, BVH_DETAIL, li.count, li.spacing, li.rotations)
		done()
	}

	return nil
}

// Will attempt to pack model's items, optimistically initially trying them all.
// If this fails after maxRetriesPerTarget attempts, use binary search to find an acceptable
// number of items to pack.
func (p *Packer) getOptimallyPackedModel() (*Model, int) {
	buildDimensions := p.config.BuildVolume
	frameSize := fauxgl.V(buildDimensions[0], buildDimensions[1], buildDimensions[2])
	// Deviation = average length of a size / 32
	// Prev comment (not sure what it means):
	//     It is not the distance between objects. And it seems that it will not
	//     reflect the distance.
	p.model.Deviation = math.Pow(p.volume, 1.0/3) / 32

	// Max retries with fresh random layouts before reducing the packing target.
	// Replaces the previous time-based limit for deterministic behaviour.
	// When annealing gets stuck quickly (dense packing), each retry can take ~20ms,
	// so 500 retries approximates the old 20s budget for fast-failing cases.
	const maxRetriesPerTarget = 500

	TRY_LIMIT := MAX_MOVE_ATTEMPTS // max number of move attempts before quitting

	// Model with max number of packed items
	bestModel := NewModel()
	bestPacked := 0

	// Binary search params - refer to number of models packed
	low := 0
	high := len(p.model.Items)
	// Optimistically set packing number as max number of items
	mid := high

	retries := 0

	//  Mesh packing loop, to find the best STL mesh packing.
	for {
		// Attempt to pack
		var iterations int
		if mid == 0 {
			// Pack will crash if mid == 0
			// Can't pack anything, so return
			break
		}
		p.model, iterations = p.model.Pack(
			ANNEALING_ITERATIONS,
			nil, // no callback
			p.sizes,
			frameSize,
			mid,
		)

		if iterations < TRY_LIMIT {
			fmt.Println("Succeeded (maybe pack more next time) ")
			fmt.Println("packing goal #, max#, min# is: ", mid, high, low)
			fmt.Println("-----------------------------------------")

			bestPacked = mid
			bestModel = p.model

			low = mid + 1
			mid = int(math.Ceil(float64((low + high) / 2)))
			retries = 0

			// Since we optimistically set mid = high, this will be true if the initial
			// run succeeds, and exit immediately as a success
			if low > high {
				break
			}
			p.model.Reset()
		} else {
			// Annealing could not find valid moves — packing is too dense
			// for this many items. Retry with a fresh random layout until
			// we exhaust the retry count for this target.
			p.model.Reset()
			retries++

			if retries >= maxRetriesPerTarget {
				fmt.Printf("Could not pack %d items after %d retries, reducing target.\n", mid, maxRetriesPerTarget)
				fmt.Println("Next packing goal # , max #, min # is: ", mid, high, low)
				fmt.Println("-----------------------------------")

				high = mid - 1
				mid = int(math.Ceil(float64((low + high) / 2)))
				retries = 0

				if low > high {
					break
				}
			}
		}

	}
	return bestModel, bestPacked
}

// Returns list of files and their transformation matrices and volumes
// Note NULL transformation implies the file was _not_ packed
func (p *Packer) generateTransformations(model *Model, itemsPacked int) ([]TransMap, float64) {
	done := timed("writing mesh")

	volume := 0.0
	transMaps := []TransMap{}

	// Note: these are the transformations applied _during the packing step_.
	// We also applied scaling and rotating (mfg) during the initialization step
	// (loadConfig). We need to reapply those here.
	transformations := model.Transformation()
	spacing := p.config.Spacing / 2.0

	for i, object := range p.objects {
		transMatrix := [4][4]float64{}
		size := p.sizes[i]
		if i < itemsPacked {
			// Model fit into packing

			// Reapply rotation and scaling
			t := transformations[i].Mul(object.mfgRotation).Mul(object.scale)

			fillVolumeWithSpacing := (size.X + spacing) * (size.Y + spacing) * (size.Z + spacing)

			volume += fillVolumeWithSpacing
			transMatrix := [4][4]float64{
				{t.X00, t.X01, t.X02, t.X03},
				{t.X10, t.X11, t.X12, t.X13},
				{t.X20, t.X21, t.X22, t.X23},
				{t.X30, t.X31, t.X32, t.X33},
			}
			transMaps = append(transMaps, TransMap{object.filename, transMatrix, fillVolumeWithSpacing})
		} else {
			// Otherwise, set as empty matrix
			transMaps = append(transMaps, TransMap{object.filename, transMatrix, 0})
		}
	}
	done()

	return transMaps, volume

}

// Attempts to pack a list of objects, applying rotation, scaling, and co-packing if specified.
func Pack(config *Config) (*PackingOutput, error) {
	// Loads files to pack into Model instance
	packer, err := NewPacker(config)
	if err != nil {
		return nil, err
	}

	// Find optimal packing orientation
	model, itemsPacked := packer.getOptimallyPackedModel()

	// Formats packed model's transformations to be exported
	transformations, volume := packer.generateTransformations(model, itemsPacked)

	meshJson, err := json.Marshal(transformations)
	if err != nil {
		return nil, err
	}

	// The JSON output always includes all items (packed and unpacked, with null
	// transformations for unpacked ones). However, the STL output is for debugging
	// and should only include the items that actually packed.
	if itemsPacked < len(model.Items) {
		model.Items = model.Items[:itemsPacked]
	}

	// Print fill ratio to stdout
	buildVolume := config.BuildVolume[0] * config.BuildVolume[1] * config.BuildVolume[2]
	fillRatio := volume / buildVolume
	fmt.Println("the fill percentage is:", fillRatio)

	// Model is for debugging (exporting STL), json is used by nautilus
	return &PackingOutput{Model: model, MeshJSON: meshJson}, nil
}
