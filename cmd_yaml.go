//
// Show a YAML file, interactively
//

package main

//
//  Entry-point.
//
func cmd_yaml(files []string) {
	for _, f := range files {
		YamlDump(f)
	}
}
