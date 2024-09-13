# Pack3d

Pack3d is a geometry packing tool for packing 3d model files on a build plate.

## Installation

See CONTRIBUTING.md

## Usage

Run `go run cmd/pack3d/main.go --help` for usage.

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



## Invoking pack3d from the command line - example

```
pack3d --input_config_json_filename=input.json --output_packing_json_filename=output
```

Notice the absence of the extension of the `output` file. This is because an `stl` file could optionally also be written as output by pack3d.

## Input example:

#### NB: the name `axes_lock` is incorrect and it stands in place of `mfg_orientation`.

```
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

## Output example (related to the input example):

1. The co-packed objects have VolumeWithSpacing = 0. This is because their volume is already contemplated in the value of the main co-packing object's VolumeWithSpacing.

2. Notice the scaling visible in the 3x3 rotation matrix.

3. pack3d can either fail to pack a set of objects entirely - an error status is displayed in the command line, or pack3d can manage to pack fewer objects in such case the objects that did not make it into the build volume will have a null Transformation = `[0, 0, 0, 0], [0, 0, 0, 0], [0, 0, 0, 0], [0, 0, 0, 1]`.


```
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
    .
    .
    .
    .
]
]
```

### Known Issues

List of issues discovered in 2024 update. These are limitations which are largely avoided if the build plate is sufficiently larger than the collective volume of items to pack.

1. If the volume of the items to pack is close to the build plate volume, sometimes pack3d will happily exceed the build plates volume.
    i. This is an issue with the internal algorithm of pack3d (i.e not Authentise code)
2. Spacing doesn't seem to be accounted for in the total volume. If I pack 2 cubes of 1x1x1 in a 2x1x1 space with 1 unit of spacing (i.e, not possible), the packing will ignore the outer bounds and pack them successfully
    i. This is because spacing is applied after packing.
3. If we can't pack all models, we use binary search to find the highest number we can. If there's a small number of models, the "candidate" in binary search might become 0. In that case, the program crashes.
    i. This makes testing for cases where we expect pack3d to fail difficult
4. For cases where all of the models will not fit, we don't appear to exit early from the packing algorithm to adjust the number of models we try to pack.
    i. This means that we will waste at least 10s trying to pack a configuration that will always fail, before we begin the binary search algo.
