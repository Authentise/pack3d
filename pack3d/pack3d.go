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

type PackingOutput struct {
	Model    *Model
	MeshJSON []byte
}

type TransMap struct {
	Filename          string
	Transformation    [4][4]float64
	VolumeWithSpacing float64
}

// Attempts to pack a list of objects, applying rotation, scaling, and co-packing if specified.
func Pack(config *Config) (*PackingOutput, error) {

	var (
		singleStlSize  []fauxgl.Vector
		scaleStl       []fauxgl.Matrix
		mfgRotationStl []fauxgl.Matrix
		done           func()
		totalVolume    float64
		iterations     int
		srcStlNames    []string
		transMaps      []TransMap
	)

	model := NewModel()
	scale := 1.0
	var scaleMatrix fauxgl.Matrix
	var mfgRotationMatrix fauxgl.Matrix

	spacing := config.Spacing / 2.0

	// frameSize is the vertex in the first quadrant
	frameSize := fauxgl.V(config.BuildVolume[0], config.BuildVolume[1], config.BuildVolume[2])
	buildVolume := config.BuildVolume[0] * config.BuildVolume[1] * config.BuildVolume[2]
	//fmt.Println(frameSize)

	/* Loading stl models */

	// Tech Debt: there's unnecessary repetition of code inside the if-statement below.

	coPackMap := make(map[string][]*Copack) // variable that contains co-packed mesh's data.
	for _, item := range config.ConfigItems {

		var mesh *fauxgl.Mesh
		var err error

		if item.Copack == nil {

			// 1. load the mesh.
			done = timed(fmt.Sprintf("loading mesh %s", item.Filename))
			mesh, err = fauxgl.LoadMesh(item.Filename)
			if err != nil {
				return nil, err
			}
			done()

			// 2. mesh centering.
			mesh.Center()

			// 3. apply the scaling to the mesh.
			//    Notice that if scaling is to be applied, it is done
			//    before the computation of the BoundingBox and volume.
			scale = item.Scale
			scaleMatrix = fauxgl.Scale(fauxgl.V(scale, scale, scale))
			if scale != 1.0 {
				done = timed("scaling mesh")
				mesh.Transform(scaleMatrix)
				done()
			}

			// 4. apply the manufacturing rotation mesh.
			//    Notice that this is done before the computation of the BoundingBox and volume.
			// IMPORTANT: do not confuse manufacturing orientation with the packing
			//            orientations from the orientations provided by the annealing further on.
			mfgRotationMatrix = item.ManufacturingOrientation()
			mesh.Transform(mfgRotationMatrix)

			// 5. update all the copies mesh for the json output.
			size := mesh.BoundingBox().Size()
			for i := 0; i < item.Count; i++ {
				singleStlSize = append(singleStlSize, size)
				srcStlNames = append(srcStlNames, item.Filename)
				scaleStl = append(scaleStl, scaleMatrix)
				mfgRotationStl = append(mfgRotationStl, mfgRotationMatrix)
			}

			fmt.Printf("== Plate == \n")
			fmt.Printf(" %g x %g y %g z\n", config.BuildVolume[0], config.BuildVolume[1], config.BuildVolume[2])

			fmt.Printf("== Mesh == \n")
			fmt.Printf("  %d triangles\n", len(mesh.Triangles))
			fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)

			// 6. coarse approx of its volume.
			totalVolume += mesh.BoundingBox().Volume()

		} else {

			coPackMap[item.Filename] = item.Copack

			// 1a. load the main co-packing mesh (the "parent" co-packing mesh, so to say).
			done = timed(fmt.Sprintf("loading the main co-packing mesh %s", item.Filename))
			mesh, err = fauxgl.LoadMesh(item.Filename)
			if err != nil {
				return nil, err
			}
			done()

			// 1b. load the co-packed meshes (the "children" of the "parent" co-packing mesh, so to say).
			for _, cp := range item.Copack {

				done = timed(fmt.Sprintf("loading the co-packed mesh %s", cp.Filename))
				coMesh, err := fauxgl.LoadMesh(cp.Filename)
				if err != nil {
					return nil, err
				}
				done()

				// add coMesh to the main mesh. The "child"'s mesh is merged into its parent's.
				mesh.Add(coMesh)
			}

			// 2. mesh centering.
			done = timed("centering co-packed mesh")
			mesh.Center()
			done()

			// 3. apply the scaling to the parent co-packing mesh (and implicitly its children).
			//    Notice that if scaling is to be applied, it is done
			//    before the computation of the BoundingBox and volume.
			scale = item.Scale
			scaleMatrix = fauxgl.Scale(fauxgl.V(scale, scale, scale))
			if scale != 1.0 {
				done = timed("scaling main co-packing mesh")
				mesh.Transform(scaleMatrix)
				done()
			}

			// 4. apply the manufacturing rotation to the parent co-packing mesh (and implicitly its children).
			//    Notice that this is done before the computation of the BoundingBox and volume.
			// IMPORTANT: do not confuse manufacturing orientation with the packing
			//            orientations from the orientations provided by the annealing further on.
			mfgRotationMatrix = item.ManufacturingOrientation()
			mesh.Transform(mfgRotationMatrix)

			// 5. update all the copies of the parent co-packing mesh
			//    (and implicitly its children) for the json output.
			size := mesh.BoundingBox().Size()
			for i := 0; i < item.Count; i++ {
				singleStlSize = append(singleStlSize, size)
				srcStlNames = append(srcStlNames, item.Filename)
				scaleStl = append(scaleStl, scaleMatrix)
				mfgRotationStl = append(mfgRotationStl, mfgRotationMatrix)
			}

			fmt.Printf("  %d triangles\n", len(mesh.Triangles))
			fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)

			// 6. coarse approx of its volume.
			totalVolume += mesh.BoundingBox().Volume()
		}

		done = timed("building bvh tree")

		model.Add(mesh, BVH_DETAIL, item.Count, spacing, item.AvailableRotations())
		done()

		fmt.Println("______________________________________________________")
	}

	side := math.Pow(totalVolume, 1.0/3)
	model.Deviation = side / 32 //it is not the distance between objects. And it seems that it will not reflect the distance.

	/*  Mesh packing loop. This loop is to find the best STL mesh packing.
	    Add 'break' in the loop to stop program */
	start := time.Now()
	maxItemNum := len(model.Items)
	var timeLimit float64
	fillVolumeWithSpacing := 0.0
	totalFillVolume := 0.0
	null := fauxgl.Matrix{
		X00: 0, X01: 0, X02: 0, X03: 0,
		X10: 0, X11: 0, X12: 0, X13: 0,
		X20: 0, X21: 0, X22: 0, X23: 0,
		X30: 0, X31: 0, X32: 0, X33: 0,
	}
	timeLimit = 20 // second

	minItemNum := 0
	packItemNum := maxItemNum
	successModel := NewModel()

	for {
		model, iterations = model.Pack(ANNEALING_ITERATIONS, nil, singleStlSize, frameSize, packItemNum)
		/* iterations is the times of trial to find a output solution, if after trying for 100 times
		   and no solution is found, then reset the model and try again. Usually if there is a solution,
		   iterations will be 1 or 2 for most cases. */
		if iterations >= 100 {
			/* There is a case that even I reset the model for many times, I still can't find a solution,
			   In this case, I need to set a threshold (20 second) to stop the software*/
			if time.Since(start).Seconds() <= timeLimit {
				model.Reset()
				continue
			} else {
				// Linear search
				//packItemNum -= 1

				// Binary search
				fmt.Println("Failed")
				fmt.Println("packing item #, max#, min# is: ", packItemNum, maxItemNum, minItemNum)
				fmt.Println("-----------------------------------")
				maxItemNum = packItemNum - 1
				packItemNum = int(math.Ceil(float64((maxItemNum + minItemNum) / 2)))

				model.Reset()
				model.Transformation()[packItemNum] = null
				start = time.Now()

				if minItemNum > maxItemNum {
					break
				}

				continue

				//TODO: Unblock the following lines if want to return a json file including the error content
				/*
					err_content := err_msg{"Cannot get a result, please decrease your numbers of STLs or enlarge the frame sizes"}
					fmt.Println(err_content.Error)
					err_json, err := json.Marshal(err_content)
					_, err := json.Marshal(err_content)
					if err != nil{
					fmt.Println("error:", err)
					}
					ioutil.WriteFile(fmt.Sprintf("%s.json", *fileNameArg), err_json, 0644)
					break
				*/
			}
		}

		// Binary search
		fmt.Println("Succeeded")
		fmt.Println("packing#, max#, min# is: ", packItemNum, maxItemNum, minItemNum)
		fmt.Println("-----------------------------------------")
		minItemNum = packItemNum + 1
		packItemNum = int(math.Ceil(float64((maxItemNum + minItemNum) / 2)))
		successModel = model
		start = time.Now()

		if minItemNum > maxItemNum {
			break
		}
		model.Reset()
	}

	done = timed("writing mesh")
	var (
		transMatrix    [4][4]float64
		fillPercentage float64
	)
	transformation := successModel.Transformation()

	// The scaling is applied directly in main.go and this is not ideal in terms of
	// modularisation but for the sake of time it had to be squished in here.
	// Tech debt: extract the scaling from main.go.
	for j := 0; j < len(successModel.Items); j++ {
		copack, ok := coPackMap[srcStlNames[j]]
		if !ok {

			t := transformation[j]
			rt := t.Mul(mfgRotationStl[j]) // manufacturing rotation for the j-th mesh.
			st := rt.Mul(scaleStl[j])      // scaled transformation for the j-th mesh.

			fillVolumeWithSpacing = (singleStlSize[j].X + spacing) * (singleStlSize[j].Y + spacing) * (singleStlSize[j].Z + spacing)
			if j < packItemNum {
				totalFillVolume += fillVolumeWithSpacing
				transMatrix = [4][4]float64{
					{st.X00, st.X01, st.X02, st.X03},
					{st.X10, st.X11, st.X12, st.X13},
					{st.X20, st.X21, st.X22, st.X23},
					{st.X30, st.X31, st.X32, st.X33},
				}
			} else {
				transMatrix = [4][4]float64{
					{0, 0, 0, 0},
					{0, 0, 0, 0},
					{0, 0, 0, 0},
					{0, 0, 0, 0},
				}
			}

			// buildVolume's filling percentage.
			fillPercentage = totalFillVolume / buildVolume

			transMaps = append(transMaps, TransMap{srcStlNames[j], transMatrix, fillVolumeWithSpacing})

		} else {

			t := transformation[j]
			rt := t.Mul(mfgRotationStl[j]) // manufacturing rotation for the j-th mesh.
			st := rt.Mul(scaleStl[j])      // scaled transformation for the j-th mesh.
			fillVolumeWithSpacing = (singleStlSize[j].X + spacing) * (singleStlSize[j].Y + spacing) * (singleStlSize[j].Z + spacing)
			if j < packItemNum {
				totalFillVolume += fillVolumeWithSpacing
				transMatrix = [4][4]float64{
					{st.X00, st.X01, st.X02, st.X03},
					{st.X10, st.X11, st.X12, st.X13},
					{st.X20, st.X21, st.X22, st.X23},
					{st.X30, st.X31, st.X32, st.X33},
				}
			} else {
				transMatrix = [4][4]float64{
					{0, 0, 0, 0},
					{0, 0, 0, 0},
					{0, 0, 0, 0},
					{0, 0, 0, 0},
				}
			}

			// buildVolume's filling percentage.
			fillPercentage = totalFillVolume / buildVolume

			// Add the main co-packing mesh to transMaps.
			transMaps = append(transMaps, TransMap{srcStlNames[j], transMatrix, fillVolumeWithSpacing})

			// Add the co-packed meshes to transMaps.
			for _, cp := range copack {
				// IMPORTANT: The volume of a co-packed object is already included in the volume
				//            of the parent object and the co-packed object's volume is set to 0.
				transMaps = append(transMaps, TransMap{cp.Filename, transMatrix, 0})
			}
		}
	}
	positionsJson, err := json.Marshal(transMaps)
	if err != nil {
		return nil, err
	}
	fmt.Println("the fill percentage is:", fillPercentage)

	done()

	return &PackingOutput{Model: successModel, MeshJSON: positionsJson}, nil
}
