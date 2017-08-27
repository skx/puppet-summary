//
// Test for our bindata.go files.
//

package main

import (
	"testing"
)

func TestAssets(t *testing.T) {

	//
	// Count of assets.
	//
	count := 0

	//
	// Iterate over each embedded asset.
	//
	for _, name := range AssetNames() {

		//
		// Get the information about the asset.
		//
		info, _ := AssetInfo(name)

		//
		// Get the actual asset-content
		//
		data, _ := Asset(name)

		if name != info.Name() {
			t.Errorf("Invalid asset-name for %s\n", name)
		}

		if int64(len(data)) != info.Size() {
			t.Errorf("Invalid asset-size for %s\n", name)
		}

		count++
	}

	names, _ := AssetDir("data")
	if len(names) != count {
		t.Errorf("Mis-matched count of assets: %d != %d\n", len(names), count)
	}
}
