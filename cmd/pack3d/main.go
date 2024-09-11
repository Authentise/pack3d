package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Authentise/pack3d/pack3d"
)

func main() {
	inputFileName := flag.String("input_config_json_filename", "", "json config file")
	outputFileName := flag.String("output_packing_json_filename", "pack3d", "export filename")
	showVersion := flag.Bool("version", false, "pack3d version")
	saveStl := flag.Bool("save_stl", false, "create stl output file")
	flag.Parse()

	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("Usage: pack3d --input_config_json_filename==mesh_config.json --output_packing_json_filename=export.json")
		fmt.Println(" - Packs N copies of each mesh into as small of a volume as possible.")
		fmt.Println(" - Runs forever, looking for the best packing.")
		fmt.Println(" - Results are written to disk whenever a new best is found.")
	}

	if *showVersion {
		fmt.Println("Pack3d 1.5.0")
		return
	}

	config, err := pack3d.ParseConfig(*inputFileName)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(config.ConfigItems) == 0 {
		flag.Usage()
		return
	}

	output, err := pack3d.Pack(config)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	saveOutput(*outputFileName, output, *saveStl)

}

func saveOutput(filename string, output *pack3d.PackingOutput, saveStl bool) {

	os.WriteFile(fmt.Sprintf("%s.json", filename), output.MeshJSON, 0644)

	// For debugging purposes, typically
	if saveStl {
		output.Model.Mesh().SaveSTL(fmt.Sprintf("%s.stl", filename))
		output.Model.TreeMesh().SaveSTL(fmt.Sprintf("%s-mesh.stl", filename))

		// When running in nautilus, we use temporary input/output files. This ensures
		// the output is always accessible
		output.Model.Mesh().SaveSTL(fmt.Sprintf("pack3d_debug_test.stl"))
	}

}
