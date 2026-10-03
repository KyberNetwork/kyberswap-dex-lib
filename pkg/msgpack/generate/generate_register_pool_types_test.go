package main

import (
	"path/filepath"
	"testing"
)

func TestImportPathsAreGoPathsOnEveryHost(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "pkg", "liquidity-source", "flywheel-fun", "pool_simulator.go")
	names, paths := getPackageNamesAndImportPaths(map[string]string{file: "PoolSimulator"}, root, "github.com/KyberNetwork/kyberswap-dex-lib")
	if names[file] != "pkg_liquiditysource_flywheelfun" {
		t.Fatalf("invalid alias %q", names[file])
	}
	expected := "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/flywheel-fun"
	if paths[expected] != file {
		t.Fatalf("invalid import path: %v", paths)
	}
}
