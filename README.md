# Pack3d

Pack3d is the geometry packing tool for 3d printing  [here](https://github.com/Authentise/pack3d). Authentise's Pack3d codebase was forked from [Fogleman's pack3d](https://github.com/fogleman/pack3d). Pack3d is written in golang and the installation instructions can be found in the CONTRIBUTING.md

## Usage

Run `go run cmd/pack3d/main.go --help` for command line help.


The most common command line usage is as:
`pack3d --input_config_json_filename=input.json --output_packing_json_filename=output`

Pack3d takes an input JSON file describing the size of a build plate, a list of items to pack, and the spacing between them.

Then pack3d will do stochastic (random re-tries) packing to pack as much as it can fit into the given build volume.

It returns a JSON file describing how the input items should be transformed to be packed, and their resulting volumes. Notice the absence of the extension of the `output` file. This is because an `stl` file could optionally also be written as output by pack3d.

## Build & Dev Tools Installation

See CONTRIBUTING.md

### Go Version Requirement

This project requires **Go 1.23.8 or later** to compile. This version requirement was updated to address CVE-2025-22871, a security vulnerability in Go's standard library `net/http` package that was fixed in Go 1.23.8. The binary must be compiled with Go 1.23.8+ to ensure the security fix is included.

### Pack3d command

Pack3d is a multi-step process:
1. Importing
    We start by loading the 3d meshes of all the input models and applying scaling and manufacturing rotation. This is distinct from the rotation the packing algorithm applies.
2. Packing
    Packing is done largely handled by the original forked code. This is done via an 'annealing' process, which tries multiple orientations and tweaking towards a minimum 'energy'.
    This process can fail. In that case, we either try just restarting the process (might have gotten stuck in a local minimum), or, if it's taken too long, we reduce the number of items to pack. We use binary search to find the maximum number of items to pack.

    **Step-size scaling** (`pack3d/anneal.go`): When packing is dense or the solver is in a shallow minimum, `DoMove` often needs many attempts to propose a valid move (most proposals are rejected for intersection, containment, or out-of-bounds). Only tiny moves succeed at the base step size, so the solver creeps slowly and may never escape. The annealer tracks how often moves "struggle" (exceed a threshold of internal attempts) over a window of 200 steps. If >= 30% of moves struggle, it temporarily increases the translation step size (`Deviation`) by 1.5x, capped at 4x the base. If <= 10% struggle, it scales back down. This adaptive behaviour helps escape dense regions and shallow minima without degrading behaviour when packing is progressing well.

    **Progress log throttling** (`pack3d/anneal.go`): The annealing loop prints progress (percentage, energy, elapsed time) to stdout. To avoid flooding the console during long runs, progress is throttled: at least 5 seconds must pass between prints, and at most 10 progress updates are shown before the final completion line.
3. Exporting
    We take the transformations of the packed items and export them to a JSON format. Note that if a model was not packed, it's transformation is a null matrix (all zeroes).

## Testing

The packing algorithm is stochastic (it uses `math/rand`). Our regression tests in `pack3d/pack3d_test.go` lock the global RNG seed so that:

- **Seed**: `testPackingInputFile` takes an optional `seed` parameter (defaults to 1). The seed ensures repeatable packing across runs. Note: `math/rand` can behave differently on different machines (OS, Go version, architecture). If tests fail with unexpected packed counts on your machine, try passing a different seed to stabilise results.

- **Output shape is stable**: the output JSON is decoded and must have `len(output) == config.TotalItems()`.
  - Note: `TotalItems()` includes `copack` entries. A config item with `count = N` and `copack` with `K` files produces `N * (1 + K)` packable items and therefore the same number of output entries.
- **Packed count is stable**: a fixture-specific `expectedPacked` value is asserted by counting how many output transformations are non-zero.

Running tests:

- **Default**: `go test ./...`
- **Short**: `go test ./... -short` (skips slow fixtures, such as `TestCh32838`)
- **Long-running**: `PACK3D_LONG_TESTS=1 go test ./...` (enables tests gated behind `PACK3D_LONG_TESTS`)

# Examples
## Example Simple Input JSON

```json

{
    "build_volume": [100, 100, 100], // Array of 3 floats
    "spacing": 5, // Float
    "items": [
        {
            "filename": "logo.stl", // Path to model file
            "count": 3, // Number of this item to pack
            "scale": 2.0, // Scale this item
            "axes_lock": { // Fixed angles if set, otherwise the packing algorithm is free to rotate models about this axis
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": null
            }, // If not supplied, treated as all values are null
            "copack": [
                {
                    "filename": "tests/jenkins_tests/corner.stl" // List of file names to copack (packed independently)
                }
            ], // If not supplied, treated as empty
        },
    ],
}
```


## Example rotation-locked Input JSON:

The name `axes_lock` (aka `mfg_orientation`) indicate of X/Y/Z need to be in an exact orientation in the final \ packed built plate. If set, that axis can't be changed during the annealing packing.

``` json
{
    "build_volume": [100, 100, 100],
    "spacing": 5,
    "items": [
        {
            "filename": "tests/jenkins_tests/logo.stl",
            "count": 3,
            "scale": 2.0,
            "axes_lock": {
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            },
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
            "axes_lock": {
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            }
        },
        {
            "filename": "tests/jenkins_tests/cube.stl",
            "count": 5,
            "scale": 1.0,
            "axes_lock": {
                "theta_x": 0.0,
                "theta_y": 0.0,
                "theta_z": 0.0
            }
        }
    ]
}
```

## Example rotation-locked output JSON:

1. `copack` items are treated as additional, independently-packable items. Each file gets its own `Transformation` and `VolumeWithSpacing`.

2. Notice the scaling is already calculated into the 4x4 translation & rotation matrix.

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
2. ~If we can't pack all models, we use binary search to find the highest number we can. If there's a small number of models, the "candidate" in binary search might become 0. In that case, the program crashes.~
3. We do not check the bounding volumes of the models we're packing compared to the build plate. This means:
    a. We will attempt to pack items which will never fit on the build plate, when we should be exiting early.
    b. We will waste attempts by packing too many items whose collective volume is greater than the build plate's.
