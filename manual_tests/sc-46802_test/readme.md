## Manual Test: no packing found and consequently not a "nested_objects" packing is achieved.

The only difference between the two input JSON file is the `build_volume` size.

### Test 01

`cd ../../..`

`/src/go/bin/pack3d --input_config_json_filename=manual_tests/sc-46802_test/input_with_insufficient_build_volume.json --output_packing_json_filename=manual_tests/sc-46802_test/output/output` 

**Result**: It should not manage to find any packing.

### Test 02

`/src/go/bin/pack3d --input_config_json_filename=manual_tests/sc-46802_test/input_with_sufficient_build_volume.json --output_packing_json_filename=manual_tests/sc-46802_test/output/output` 

**Result**: This one instead should manage to find a packing.
