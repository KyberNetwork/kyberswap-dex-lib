package main

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	embedLine    = regexp.MustCompile(`^//go:embed (\S+\.json)$`)
	generateLine = regexp.MustCompile(`^//go:generate go run github\.com/KyberNetwork/kyberswap-dex-lib/` +
		regexp.QuoteMeta(dir) + ` (\w+) (\S+\.json)$`)
)

// TestEveryContractFileIsGenerated: every contract file a package embeds, {"abi", "bytecode"}, has
// the go:generate that writes it right above its go:embed, and the contract it names is declared in
// one file here, so go generate rebuilds every one and nothing else. It needs no forge.
func TestEveryContractFileIsGenerated(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	contracts := filepath.Join(root, dir)
	generated := 0
	err = filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		var previous string
		for lines := bufio.NewScanner(f); lines.Scan(); previous = lines.Text() {
			embed := embedLine.FindStringSubmatch(lines.Text())
			if embed == nil || !isContract(filepath.Join(filepath.Dir(path), embed[1])) {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			generate := generateLine.FindStringSubmatch(previous)
			if generate == nil || generate[2] != embed[1] {
				t.Errorf("%s: no go:generate writes %s above its go:embed", rel, embed[1])
				continue
			}
			if _, err := source(contracts, generate[1]); err != nil {
				t.Errorf("%s: %v", rel, err)
			}
			generated++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if generated == 0 {
		t.Fatal("no contract file found")
	}
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
