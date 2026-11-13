package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const rebuild = "run go generate -run ephemeral/contracts ./pkg/..."

var (
	embedLine    = regexp.MustCompile(`^//go:embed (\S+\.json)$`)
	generateLine = regexp.MustCompile(`^//go:generate go run github\.com/KyberNetwork/kyberswap-dex-lib/` +
		regexp.QuoteMeta(dir) + ` (\w+) (\S+\.json)$`)
)

// TestEveryContractFileIsGenerated: every contract file a package embeds, {"abi", "bytecode"}, has
// the go:generate that writes it right above its go:embed, and the contract it names is declared in
// one file here, so go generate rebuilds every one and nothing else. It needs no forge.
func TestEveryContractFileIsGenerated(t *testing.T) {
	root := testRoot(t)
	for file, name := range generated(t, root) {
		if _, err := source(filepath.Join(root, dir), name); err != nil {
			t.Errorf("%s: %v", file, err)
		}
	}
}

// TestContractFilesMatchTheirSources: every contract file is what go generate last wrote, from the
// Solidity and foundry.toml here as they are now, by the SHA-256 it recorded of each in
// sources.json; and every Solidity file here goes into one. A file edited by hand, a contract
// changed without a rebuild, or one never built fails it. It needs no forge.
func TestContractFilesMatchTheirSources(t *testing.T) {
	root := testRoot(t)
	manifest, err := readManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	files := generated(t, root)
	for file := range manifest {
		if _, ok := files[file]; !ok {
			t.Errorf("%s is in %s but no go:generate writes it; drop it from there", file, manifestFile)
		}
	}

	sums := map[string]string{} // each file hashed once, however many contracts it goes into
	sumOf := func(path string) string {
		if s, ok := sums[path]; ok {
			return s
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Error(err)
		}
		sums[path] = sum(data)
		return sums[path]
	}
	stale := map[string][]string{} // each changed source, with the contract files built from it
	built := map[string]bool{}
	for file, name := range files {
		e, ok := manifest[file]
		if !ok || e.Contract != name {
			t.Errorf("%s was not built from %s, %s", file, name, rebuild)
			continue
		}
		if sumOf(file) != e.SHA256 {
			t.Errorf("%s was changed since go generate wrote it; do not edit it by hand, %s", file, rebuild)
		}
		for src, want := range e.Sources {
			built[src] = true
			if sumOf(src) != want {
				stale[src] = append(stale[src], file)
			}
		}
	}
	for _, src := range slices.Sorted(maps.Keys(stale)) {
		slices.Sort(stale[src])
		t.Errorf("%s changed since %s were built from it, %s", src, strings.Join(stale[src], ", "), rebuild)
	}

	sol, err := solidity(filepath.Join(root, dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range sol {
		if path := dir + "/" + file; !built[path] {
			t.Errorf("%s goes into no contract file; build it into one, or remove it", path)
		}
	}
}

func testRoot(t *testing.T) string {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// generated is every contract file a package embeds, by its path from root, with the contract that
// its go:generate line writes into it. A contract file without that line is an error of t.
func generated(t *testing.T, root string) map[string]string {
	files := map[string]string{}
	err := filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.Contains(raw, []byte("//go:embed ")) {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		var previous string
		for line := range strings.Lines(string(raw)) {
			line = strings.TrimRight(line, "\r\n")
			embed := embedLine.FindStringSubmatch(line)
			if embed != nil && isContract(filepath.Join(filepath.Dir(path), embed[1])) {
				generate := generateLine.FindStringSubmatch(previous)
				if generate == nil || generate[2] != embed[1] {
					t.Errorf("%s: no go:generate writes %s above its go:embed", rel, embed[1])
				} else {
					files[filepath.ToSlash(filepath.Join(filepath.Dir(rel), embed[1]))] = generate[1]
				}
			}
			previous = line
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no contract file found")
	}
	return files
}

// isContract reports whether path, a file and not a pattern, is a contract file: an object of abi
// and bytecode alone.
func isContract(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil || len(doc) != 2 {
		return false
	}
	_, abi := doc["abi"]
	_, bytecode := doc["bytecode"]
	return abi && bytecode
}
