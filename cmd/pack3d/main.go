/*
Instruction: write down this into the command line to use the software.

<pack3d --input_config_json_filename=json_config_file --output_packing_json_filename=export_filename>
For example: <pack3d --input_config_json_filename=input.json>

The frame and spacing's units, in the json file, are in millimeters.
*/

package main

import (
	"flag"
	"fmt"

	"github.com/Authentise/pack3d/pack3d"
)

func main() {
	jsonFileArg := flag.String("input_config_json_filename", "", "json config file")
	fileNameArg := flag.String("output_packing_json_filename", "pack3d", "export filename")
	versionArg := flag.Bool("version", false, "pack3d version")
	flag.Parse()

	if *versionArg {
		fmt.Println("Pack3d 1.5.0")
		return
	}


	pack3d.Pack(*jsonFileArg, *fileNameArg)
}

