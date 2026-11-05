// Command contracts builds a contract from the Solidity in this directory and writes it to the
// JSON file a tracker embeds, as {"abi": ..., "bytecode": ...}. Each package runs it through
// go generate, next to the file's go:embed:
//
//	//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts V3Lens abis/lens.json
//
// It needs forge (Foundry). Without it, it leaves the file as it is and says so, so go generate
// needs nothing but Go. With it, it builds by foundry.toml here, which fixes the compiler and its
// settings and drops the metadata hash, so a build gives the same bytecode on every machine.
//
// It records in sources.json the SHA-256 of the file it wrote and of every file the contract was
// built from, which a test checks without forge: a contract file edited by hand, or one whose
// Solidity changed without a rebuild, fails it.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
)

// dir is this directory, relative to the module root.
const dir = "pkg/util/ephemeral/contracts"

// manifestFile, in dir, records what each contract file was built from.
const manifestFile = "sources.json"

// entry is a contract file in the manifest, which keys it by its path from the module root, as it
// keys its sources.
type entry struct {
	Contract string            `json:"contract"`
	SHA256   string            `json:"sha256"`
	Sources  map[string]string `json:"sources"` // SHA-256 of foundry.toml and each Solidity file
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: contracts <Contract> <file.json>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "contracts:", err)
		os.Exit(1)
	}
}

func run(name, out string) error {
	forge, err := exec.LookPath("forge")
	if err != nil {
		fmt.Fprintf(os.Stderr, "contracts: forge not found; %s left as is\n", out)
		return nil
	}
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	contracts := filepath.Join(root, dir)
	src, err := source(contracts, name)
	if err != nil {
		return err
	}

	build := exec.Command(forge, "build", "--root", contracts, src)
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("forge build %s: %w", src, err)
	}
	artifact, err := os.ReadFile(filepath.Join(contracts, "out", filepath.Base(src), name+".json"))
	if err != nil {
		return err
	}
	content, err := render(artifact)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if wrote, err := writeIfChanged(out, content); err != nil {
		return err
	} else if wrote {
		fmt.Fprintf(os.Stderr, "contracts: wrote %s (%s)\n", out, name)
	}
	return record(root, out, name, content, artifact)
}

// record sets out's entry in the manifest: name, the SHA-256 of content, and that of foundry.toml
// and of every source the artifact lists.
func record(root, out, name string, content, artifact []byte) error {
	var built struct {
		Metadata struct {
			Sources map[string]json.RawMessage `json:"sources"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(artifact, &built); err != nil {
		return err
	}
	if len(built.Metadata.Sources) == 0 {
		return fmt.Errorf("%s: the artifact lists no sources", name)
	}
	e := entry{Contract: name, SHA256: sum(content), Sources: map[string]string{}}
	for _, src := range append(slices.Sorted(maps.Keys(built.Metadata.Sources)), "foundry.toml") {
		path := filepath.ToSlash(filepath.Join(dir, src))
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return err
		}
		e.Sources[path] = sum(data)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	key, err := filepath.Rel(root, abs)
	if err != nil {
		return err
	}

	manifest, err := readManifest(root)
	if err != nil {
		return err
	}
	manifest[filepath.ToSlash(key)] = e
	content, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	_, err = writeIfChanged(filepath.Join(root, dir, manifestFile), append(content, '\n'))
	return err
}

// writeIfChanged writes content to path unless path holds it already, and reports whether it wrote.
func writeIfChanged(path string, content []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, content) {
		return false, nil
	}
	return true, os.WriteFile(path, content, 0o644)
}

// readManifest is the manifest under root, empty when there is none yet.
func readManifest(root string) (map[string]entry, error) {
	manifest := map[string]entry{}
	raw, err := os.ReadFile(filepath.Join(root, dir, manifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest, nil
	} else if err != nil {
		return nil, err
	}
	return manifest, json.Unmarshal(raw, &manifest)
}

func sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// moduleRoot is the directory of the go.mod at or above the working directory.
func moduleRoot() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", errors.New("no go.mod at or above the working directory")
		}
		d = parent
	}
}

// solidity is every Solidity file under contracts, relative to it, forge's out and cache left out.
func solidity(contracts string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(contracts, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && (d.Name() == "out" || d.Name() == "cache"):
			return filepath.SkipDir
		case !d.IsDir() && filepath.Ext(path) == ".sol":
			rel, _ := filepath.Rel(contracts, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files, err
}

// source is the file under contracts, relative to it, that declares contract name.
func source(contracts, name string) (string, error) {
	files, err := solidity(contracts)
	if err != nil {
		return "", err
	}
	declares := regexp.MustCompile(`(?m)^\s*(?:abstract\s+)?contract\s+` + regexp.QuoteMeta(name) + `\b`)
	var found []string
	for _, file := range files {
		text, err := os.ReadFile(filepath.Join(contracts, file))
		if err != nil {
			return "", err
		}
		if declares.Match(text) {
			found = append(found, file)
		}
	}
	if len(found) != 1 {
		return "", fmt.Errorf("contract %s is declared in %d files under %s, not one: %v", name, len(found), dir,
			found)
	}
	return found[0], nil
}

// render is the file a tracker embeds from forge's artifact: the ABI and the creation code, indented
// with two spaces and ending in a newline, as a release writes it.
func render(artifact []byte) ([]byte, error) {
	var built struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode struct {
			Object string `json:"object"`
		} `json:"bytecode"`
	}
	if err := json.Unmarshal(artifact, &built); err != nil {
		return nil, err
	}
	if len(built.ABI) == 0 || len(built.Bytecode.Object) <= len("0x") {
		return nil, errors.New("the artifact holds no ABI or no creation code")
	}
	// Decoded into maps, which encoding/json writes with sorted keys, as releases have them; forge
	// writes them in an order of its own.
	var abi any
	dec := json.NewDecoder(bytes.NewReader(built.ABI))
	dec.UseNumber()
	if err := dec.Decode(&abi); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(struct {
		ABI      any    `json:"abi"`
		Bytecode string `json:"bytecode"`
	}{abi, built.Bytecode.Object}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}
