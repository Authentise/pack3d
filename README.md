# pack3d

Pack3d is the geometry packing tool for 3d printing  [here](https://github.com/Authentise/pack3d). Authentise's Pack3d codebase was forked from [Fogleman's pack3d](https://github.com/fogleman/pack3d). Pack3d is written in golang and the installation instructions can be found in the CONTRIBUTING.md

Pack3d takes STL files and a JSON file of layout limits / complexities, and does stochastic (random re-tries) packing to pack as much as it can 
into the given build volume. 

STL files do *not* have units, nor does this tool. Assume mm, but size are in un-named 'STL Units'.

## Invoking pack3d from the command line - example
Jump-start using this tool like this:
```
pack3d --input_config_json_filename=input.json --output_packing_json_filename=output --save_stl
```

Notice the absence of the extension of the `output` file. This will output Mesh, STL, and 'json of meta-data' files based on that bsaename.  

## Input example(s):
See folder `tests/jenkins_tests/input_$NAME` for examples of use. 


### Key features of input json file
 - the name `axes_lock` indicated if a model has a locked packing orientation, aka (`mfg_orientation`). Null indicates it can be rotated in that 
direction by the packing tool 
 - 'axes_locked' is required, some (most?) keys are required per `item`.
 - The 'spacing' is minimum distance between objects as you pack them 
 - The 'co-packing' is if 2 STL geometries touch / print touching in a locked orientation
 - Scaling needs to be set especially of model STL's are in different units from the build-space outline

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
