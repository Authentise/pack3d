package pack3d

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

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

	// Empty if no copacking occured
	copackedFiles []string
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

func (p *Packer) loadConfig(config *Config) error {
	// Initialize packer fields
	p.config = config
	p.model = NewModel()
	// Capacity of number of items
	p.objects = make([]Object, 0, len(config.ConfigItems))
	p.sizes = make([]fauxgl.Vector, 0, len(config.ConfigItems))

	for _, item := range config.ConfigItems {
		object := Object{}
		// 1. load the mesh.
		done := timed(fmt.Sprintf("loading mesh %s", item.Filename))
		mesh, err := fauxgl.LoadMesh(item.Filename)
		if err != nil {
			return err
		}
		done()
		if item.Copack != nil {
			copackedFiles := []string{}
			for _, cp := range item.Copack {
				done = timed(fmt.Sprintf("loading the co-packed mesh %s", cp.Filename))
				coMesh, err := fauxgl.LoadMesh(cp.Filename)
				if err != nil {
					return err
				}
				done()

				// add coMesh to the main mesh. The "child"'s mesh is merged into its parent's.
				mesh.Add(coMesh)
				copackedFiles = append(copackedFiles, cp.Filename)
			}
			object.copackedFiles = copackedFiles
		}
		// 2. mesh centering.
		mesh.Center()

		// 3. apply the scaling to the mesh.
		//    Notice that if scaling is to be applied, it is done
		//    before the computation of the BoundingBox and volume.
		object.scale = fauxgl.Scale(fauxgl.V(item.Scale, item.Scale, item.Scale))
		if item.Scale != 1.0 {
			done = timed("scaling mesh")
			mesh.Transform(object.scale)
			done()
		}

		// 4. apply the manufacturing rotation mesh.
		//    Notice that this is done before the computation of the BoundingBox and volume.
		// IMPORTANT: do not confuse manufacturing orientation with the packing
		//            orientations from the orientations provided by the annealing further on.
		object.mfgRotation = item.ManufacturingOrientation()
		mesh.Transform(object.mfgRotation)

		// 5. update all the copies mesh for the json output.
		size := mesh.BoundingBox().Size()
		object.filename = item.Filename

		for range item.Count {
			p.objects = append(p.objects, object)
			p.sizes = append(p.sizes, size)
		}

		fmt.Printf("  %d triangles\n", len(mesh.Triangles))
		fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)

		// 6. coarse approx of its volume.
		p.volume += mesh.BoundingBox().Volume()

		done = timed("building bvh tree")

		p.model.Add(mesh, BVH_DETAIL, item.Count, config.Spacing/2, item.AvailableRotations())
		done()

		fmt.Println("______________________________________________________")

	}

	return nil
}

// Will attempt to pack model's items, optimistically initially trying them all
// If this fails, and takes long (>10s), use binary search to find an acceptable
// number of items to pack
func (p *Packer) getOptimallyPackedModel() (*Model, int) {
	buildDimensions := p.config.BuildVolume
	frameSize := fauxgl.V(buildDimensions[0], buildDimensions[1], buildDimensions[2])
	// Deviation = average length of a size / 32
	// Prev comment (not sure what it means):
	//     It is not the distance between objects. And it seems that it will not
	//     reflect the distance.
	p.model.Deviation = math.Pow(p.volume, 1.0/3) / 32

	start := time.Now()
	TIME_LIMIT:= 10.0 //10 seconds per Stochastic try , then start again
	TRY_LIMIT := 100 // max number of Stochastic tries before quitting

	// Model with max number of packed items
	bestModel := NewModel()
	bestPacked := 0

	// Binary search params - refer to number of models packed
	low := 0
	high := len(p.model.Items)
	// Optimistically set packing number as max number of items
	mid := high

	//  Mesh packing loop, to find the best STL mesh packing.
	for {
		// Attempt to pack
		iterations := 0
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

		// Iterations < 100 considered successful
		if iterations <  TRY_LIMIT {
			fmt.Println("Succeeded (maybe pack more next time) ")
			fmt.Println("packing goal #, max#, min# is: ", mid, high, low)
			fmt.Println("-----------------------------------------")

			bestPacked = mid
			bestModel = p.model

			//  if success, 'bisect' extend "models to pack" count 
			low = mid + 1
			mid = int(math.Ceil(float64((low + high) / 2)))
			start = time.Now()

			// Since we optimistically set mid = high, this will be true if the initial
			// run succeeds, and exit immedately as a success
			if low > high {
				break
			}
			p.model.Reset()
		} else {
			// If iterations > 100, we consider this as failed. Should take 1-2 iterations
			p.model.Reset()
			
			fmt.Println("Iterations > 100. Failed (maybe pack fewer next time)")
			if time.Since(start).Seconds() > TIME_LIMIT {
				//  if failed, and past time limit, 'bisect' shrink "models to pack" count 
				fmt.Println("Next packing goal # , max #, min # is: ", mid, high, low)
				fmt.Println("-----------------------------------")

				// Binary search for lower packing number
				high = mid - 1
				mid = int(math.Ceil(float64((low + high) / 2)))

				// This array is a copy, this shouldn't do anything?
				p.model.Transformation()[mid] = NULL_TRANSFORMATION
				// Reset initial start time
				start = time.Now()

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

		// Add the co-packed meshes to transMaps.
		for _, filename := range object.copackedFiles {
			// Volume = 0 since it's already included in the parent
			transMaps = append(transMaps, TransMap{filename, transMatrix, 0})
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

	// Print fill ratio to stdout
	buildVolume := config.BuildVolume[0] * config.BuildVolume[1] * config.BuildVolume[2]
	fillRatio := volume / buildVolume
	fmt.Println("the fill percentage is:", fillRatio)

	// Model is for debugging (exporting STL), json is used by nautilus
	return &PackingOutput{Model: model, MeshJSON: meshJson}, nil
}
