package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	poolSimFileName    = "pool_simulator.go"
	moduleName         = "github.com/KyberNetwork/kyberswap-dex-lib"
	regularPoolSimName = "PoolSimulator"
)

var (
	irregularPoolSimNameByPackageName = map[string]string{
		"pkg_source_balancer_stable":   "StablePool",
		"pkg_source_balancer_weighted": "WeightedPool2Tokens",
		"pkg_source_curve_aave":        "AavePool",
		"pkg_source_curve_base":        "PoolBaseSimulator",
		"pkg_source_curve_compound":    "CompoundPool",
		"pkg_source_curve_meta":        "Pool",
		"pkg_source_curve_plainoracle": "Pool",
		"pkg_source_curve_tricrypto":   "Pool",
		"pkg_source_curve_two":         "Pool",
		"pkg_source_maverickv1":        "Pool",
		"pkg_source_velocimeter":       "Pool",
	}
)

func main() {
	var paths []string

	if dir := findGoModDirInParents(); dir != "" {
		for _, path := range findAllPoolTestdataSourceFile(dir) {
			path = strings.TrimPrefix(path, dir+"/")
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return
	}

	pkgNames := getPackageNamesFromSourceFiles(paths)
	importPaths := getPackageImportPathsFromSourceFiles(paths)

	outFile, err := os.Create("./register_pool_types.go")
	if err != nil {
		log.Fatalf("could not create dispatch_gen.go: %s", err)
	}
	defer outFile.Close()

	outFileBuf := bufio.NewWriter(outFile)
	defer outFileBuf.Flush()

	emitImports(outFileBuf, pkgNames, importPaths)

	fmt.Fprintf(outFileBuf, "func init() {\n")
	for _, pkgName := range pkgNames {
		poolSimName := regularPoolSimName
		if name, ok := irregularPoolSimNameByPackageName[pkgName]; ok {
			poolSimName = name
		}
		fmt.Fprintf(outFileBuf, "\tmsgpack.RegisterConcreteType(&%s.%s{})\n", pkgName, poolSimName)
	}
	fmt.Fprintf(outFileBuf, "}\n")

}

func emitImports(outFileBuf io.Writer, pkgNames, importPaths []string) {
	fmt.Fprintf(outFileBuf, "package msgpack\n")
	fmt.Fprintf(outFileBuf, "\n")

	fmt.Fprintf(outFileBuf, "// Code generated by %s/pkg/msgpack/generate DO NOT EDIT.\n", moduleName)
	fmt.Fprintf(outFileBuf, "\n")

	fmt.Fprintf(outFileBuf, "import (\n")
	fmt.Fprintf(outFileBuf, "\t\"github.com/KyberNetwork/msgpack/v5\"\n")
	fmt.Fprintf(outFileBuf, "\n")
	for i, dexName := range pkgNames {
		fmt.Fprintf(outFileBuf, "\t%s \"%s\"\n", dexName, importPaths[i])
	}
	fmt.Fprintf(outFileBuf, ")\n")
}

func findGoModDirInParents() string {
	var (
		hasGoMod = false
		cwd, _   = os.Getwd()
		visited  = make(map[string]struct{}) // to eliminate cycle
	)
	for {
		if _, ok := visited[cwd]; ok {
			break
		}
		visited[cwd] = struct{}{}

		entries, err := os.ReadDir(cwd)
		if err != nil {
			break
		}
		for _, entry := range entries {
			if entry.Name() == "go.mod" {
				hasGoMod = true
				break
			}
		}
		if hasGoMod {
			break
		}

		cwd = filepath.Join(cwd, "..")
		cwd, err = filepath.Abs(cwd)
		if err != nil {
			break
		}
	}
	if hasGoMod {
		return cwd
	}
	return ""
}

func findAllPoolTestdataSourceFile(rootDir string) []string {
	var paths []string
	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "/"+poolSimFileName) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return paths
}

func getPackageNamesFromSourceFiles(sourcePaths []string) []string {
	importNames := make([]string, 0, len(sourcePaths))
	for _, path := range sourcePaths {
		dexName := strings.TrimSuffix(path, "/"+poolSimFileName)
		dexName = strings.ReplaceAll(dexName, "-", "")
		dexName = strings.ReplaceAll(dexName, "/", "_")
		importNames = append(importNames, dexName)
	}
	return importNames
}

func getPackageImportPathsFromSourceFiles(sourcePaths []string) []string {
	paths := make([]string, 0, len(sourcePaths))
	for _, path := range sourcePaths {
		importPath := filepath.Join(moduleName, strings.TrimSuffix(path, "/"+poolSimFileName))
		paths = append(paths, importPath)
	}
	return paths
}
