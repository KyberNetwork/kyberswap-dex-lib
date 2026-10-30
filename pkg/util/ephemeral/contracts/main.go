// Command contracts builds a contract from the Solidity in this directory and writes it to the
// JSON file a tracker embeds, as {"abi": ..., "bytecode": ...}. Each package runs it through
// go generate, next to the file's go:embed:
//
//	//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts V3Lens abis/lens.json
//
// It needs forge (Foundry). Without it, it leaves the file as it is and says so, so go generate
// needs nothing but Go. With it, it builds by foundry.toml here, which fixes the compiler and its
// settings and drops the metadata hash, so a build gives the same bytecode on every machine.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// dir is this directory, relative to the module root.
const dir = "pkg/util/ephemeral/contracts"

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
	if old, err := os.ReadFile(out); err == nil && bytes.Equal(old, content) {
		return nil
	}
	if err := os.WriteFile(out, content, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "contracts: wrote %s (%s)\n", out, name)
	return nil
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

// source is the file under contracts, relative to it, that declares contract name.
func source(contracts, name string) (string, error) {
	declares := regexp.MustCompile(`(?m)^\s*(?:abstract\s+)?contract\s+` + regexp.QuoteMeta(name) + `\b`)
	var found []string
	err := filepath.WalkDir(contracts, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "out" || d.Name() == "cache") {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".sol" {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if declares.Match(text) {
			rel, _ := filepath.Rel(contracts, path)
			found = append(found, rel)
		}
		return nil
	})
	if err != nil {
		return "", err
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
