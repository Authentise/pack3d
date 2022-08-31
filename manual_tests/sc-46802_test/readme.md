## Manual Test: no packing found and consequently not a "nested_objects" packing is achieved.
 
**Test**
cd ../../..
 <COMMAND TO RUN> `/src/go/bin/pack3d -input_config_json_filename manual_tests/partially_successful_test/input.json -output_packing_json_filename manual_tests/partially_successful_test/output/output` 

Result: It should not manage to fit all objects in.