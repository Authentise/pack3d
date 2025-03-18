import os
from subprocess import check_call
from os import environ



print(os.listdir())
check_call(["./main", "--input_config_json_filename","input.json", "--output_packing_json_filename", "out"],  env=environ)
