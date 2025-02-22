# Pack3d

Pack3d is the geometry packing tool for 3d printing  [here](https://github.com/Authentise/pack3d). Authentise's Pack3d codebase was forked from [Fogleman's pack3d](https://github.com/fogleman/pack3d). Pack3d is written in golang and the installation instructions can be found in the CONTRIBUTING.md

Pack3d takes a JSON file of what STL files need to be packed, and returns a JSON file of what was possible to pack out of that. 

The JSON file gives targets, build size, count of items, and limits / complexities of packing. Then pack3d will do stochastic (random re-tries) packing to pack as much as it can fit into the given build volume. 

## Installation
To build binary to run in nautils (flows backend) see CONTRIBUTING.md

## Debugging Build & Run 

To create a binary for testing , run:
 `go build -o <output path> cmd/pack3d/main.go`
Pack3d may build a number of binaries, found in `/cmd` folder. Of these, only `pack3d` is currently used.


For usage info: 
 `go run cmd/pack3d/main.go --help` 

To run the default tests (it mauy take several minutes): 
 `go test ./... ` 
 This may 

 To run a single test, by function name 
```
go test --run TestZeroPackingUntilPackFails -v  ./pack3d/pack3d_test.go
```

Invoking pack3d from the command line example:
```
pack3d --input_config_json_filename=input.json --output_packing_json_filename=output
```
Notice the absence of the extension of the `output` file. This is because an `stl` file could optionally also be written as output by pack3d, 
alongside the  JSON output.


### Pack3d Process 

Pack3d is a multi-step process:
1. Importing
    Pack3d loads the 3d meshes of all the input models and applying scaling and manufacturing rotation. Some STL files are relative to the other, and will rotate / pack togehter as a grouping. This step is *distinct from* the rotation the packing algorithm applies later.

2. Packing
    Packing is largely handled by the original forked code. This is done via an 'annealing' process, which tries multiple orientations and tweaking towards a minimum 'energy'.
    
    The annealing process can fail. In that case, we either try just restarting the process (might have gotten stuck in a local minimum), or, if it's taken too long, we reduce the number of items to pack. We use binary search to find the maximum number of items to pack. In case of a crash, it may use a previous less-good packing as a fallback.

3. Exporting
    We take the transformations of the packed items and export them to a JSON format. Note that if a model was not packed, it's transformation is a null matrix (all zeroes).

### Known Issues

List of issues discovered in 2024 update. These are limitations which are largely avoided if the build plate is sufficiently larger than the collective volume of items to pack.

1. If the volume of the items to pack is close to the build plate volume, sometimes pack3d will silently exceed the build plates volume.
    i. This is an issue with the internal algorithm of pack3d (i.e not Authentise code)

2. ~If we can't pack all models, we use binary search to find the highest number we can. If there's a small number of models, the "candidate" in binary search might become 0. In that case, the program crashes.~

3. We do not check the bounding volumes of the models we're packing compared to the build plate. This means:
    a. We will attempt to pack items which will never fit on the build plate, when we should be exiting early.
    b. We will waste attempts by packing too many items whose collective volume is greater than the build plate's.

# Examples

## Simple Input JSON Example

Simple Example input JSON with notes 

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


## Complex Input JSON example:

The name `axes_lock` (aka `mfg_orientation`) indicate of X/Y/Z need to be in an exact orientation in the final \
packed built plate. If set, that axis can't be changed during the annealing packing.

```json
{
    "build_volume": [100, 100, 100],
    "spacing": 5,
    "items": [
        {
            "filename": "tests/jenkins_tests/logo.stl",
            "count": 3,
            "scale": 2.0,
            "axes_lock": [
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            ],
            "copack": [
                {
                    "filename": "tests/jenkins_tests/corner.stl"
                }
            ]
        },
        {
            "filename": "tests/jenkins_tests/cube.stl",
            "count": 2,
            "scale": 4.0,
            "axes_lock": [
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            ],
        },
        {
            "filename": "tests/jenkins_tests/cube.stl",
            "count": 5,
            "scale": 1.0,
            "axes_lock": [
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            ],
        }
    ]
}
```

## Complex Output JSON Example:
This complex output is related to the input example. 

1. The co-packed objects have VolumeWithSpacing = 0. This is because their volume is already contemplated in the value of the main co-packing object's VolumeWithSpacing.

2. Notice the scaling is already built into the 3x3 rotation matrix.

1. pack3d can either fail to pack a set of objects entirely - an error status is displayed in the command line, or pack3d can manage to pack fewer objects in such case the objects that did not make it into the build volume will have a null Transformation = `[0, 0, 0, 0], [0, 0, 0, 0], [0, 0, 0, 0], [0, 0, 0, 1]`.


```json
[
    {
        "Filename": "tests/jenkins_tests/logo.stl",
        "Transformation": [
            [ 0, 0, -2, -36.092921290618406],
            [-2, 0,  0, -4.735731505145346],
            [ 0, 2,  0, 8.056191563929794],
            [ 0, 0, 0, 1]
        ],
        "VolumeWithSpacing": 5138.241184594143
    },
    {
        "Filename": "tests/jenkins_tests/logo.stl",
        "Transformation": [
            [ 0, 0, 2, -36.09345621544282],
            [ 2, 0, 0, -21.623185522659124],
            [ 0, 2, 0, -8.785596546107582],
            [ 0, 0, 0, 1]
        ],
        "VolumeWithSpacing": 5138.241184594143
    },
    {
        "Filename": "tests/jenkins_tests/cube.stl",
        "Transformation": [
            [ 0, 4, 0, -4.439943270386402],
            [ 0, 0, 4, -11.647772698025165],
            [ 4, 0, 0, -28.684157525681382],
            [ 0, 0, 0, 1]
        ],
        "VolumeWithSpacing": 76765.625
    },
]
```

