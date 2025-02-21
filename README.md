# Pack3d

Pack3d is the geometry packing tool for 3d printing  [here](https://github.com/Authentise/pack3d). Authentise's Pack3d codebase was forked from [Fogleman's pack3d](https://github.com/fogleman/pack3d). Pack3d is written in golang and the installation instructions can be found in the CONTRIBUTING.md

Pack3d takes STL files and a JSON file of layout limits / complexities, and does stochastic (random re-tries) packing to pack as much as it can 
into the given build volume. 

## Installation

See CONTRIBUTING.md

## Usage

Run `go run cmd/pack3d/main.go --help` for usage.

## Build

To create a binary, run:

`go build -o <output path> cmd/pack3d/main.go`

To tag it with the current commit, run:

`go build -o bin/pack3d-$(git rev-parse --short HEAD) cmd/pack3d/main.go`

## Overview

Pack3d consists of a number of binaries, found in `/cmd` folder. Of these, only `pack3d` is currently used.

### Pack3d command

Pack3d takes an input JSON file describing the size of a build plate, a list of items to pack, and the spacing between them. It returns a JSON file describing how the input items should be transformed to be packed, and their resulting volumes.

Pack3d is a multi-step process:
1. Importing
    We start by loading the 3d meshes of all the input models and applying scaling and manufacturing rotation. This is distinct from the rotation the packing algorithm applies.
2. Packing
    Packing is done largely handled by the original forked code. This is done via an 'annealing' process, which tries multiple orientations and tweaking towards a minimum 'energy'.
    This process can fail. In that case, we either try just restarting the process (might have gotten stuck in a local minimum), or, if it's taken too long, we reduce the number of items to pack. We use binary search to find the maximum number of items to pack.
3. Exporting
    We take the transformations of the packed items and export them to a JSON format. Note that if a model was not packed, it's transformation is a null matrix (all zeroes).

### Input Schema

```json

{
    "build_volume": [100, 100, 100], // Array of 3 floats
    "spacing": 5, // Float
    "items": [
        {
            "filename": "logo.stl", // Path to model file
            "count": 3, // Number of this item to pack
            "scale": 2.0, // Scale this item
            "axes_lock": [ // Fixed angles if set, otherwise the packing algorithm is free to rotate models about this axis
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": null
            ], // If not supplied, treated as all values are null
            "copack": [
                {
                    "filename": "tests/jenkins_tests/corner.stl" // List of file names to copack
                }
            ], // If not supplied, treated as empty
        },
    ],
}
```
