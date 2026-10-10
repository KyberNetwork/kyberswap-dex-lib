package lotflow

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// F6: the JSON fixtures are stored gzipped; every loader takes the .json path and reads the .json.gz
// beside it transparently.
func TestReadJSONTransparentGzip(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "index.json.gz"))
	require.NoError(t, err)
	zw := gzip.NewWriter(f)
	_, err = zw.Write([]byte(`{"schema":"s","cases":[{"id":"a","file":"a.json"}]}`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())
	for _, p := range []string{"index.json", "index.json.gz"} {
		idx, err := LoadIndex(filepath.Join(dir, p))
		require.NoError(t, err, p)
		require.Len(t, idx.Cases, 1, p)
	}
	_, err = LoadIndex(filepath.Join(dir, "missing.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

// fixturePaths lists every case file under testdata/<suite>/ by its .json name, whether it is stored
// plain or gzipped.
func fixturePaths(t testing.TB, suite string) []string {
	var out []string
	seen := map[string]bool{}
	for _, pat := range []string{"*.json", "*.json.gz"} {
		ps, err := filepath.Glob(filepath.Join("testdata", suite, pat))
		require.NoError(t, err)
		for _, p := range ps {
			p = strings.TrimSuffix(p, ".gz")
			if !seen[p] && filepath.Base(p) != "index.json" {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	return out
}

// TestFixtureCorpusDigest prints, per suite, the case count and a digest of every case's decoded
// bytes and replay result (Quote and QuoteConservative). With LOTFLOW_DIGEST_OUT set it also writes
// the per-case lines, so a storage change can be proven to keep every case and every result.
func TestFixtureCorpusDigest(t *testing.T) {
	var lines []string
	for _, suite := range []string{"fixtures", "a5", "mq", "parity"} {
		h := sha256.New()
		n := 0
		for _, p := range fixturePaths(t, suite) {
			b, err := readFile(p)
			require.NoError(t, err, p)
			line := fmt.Sprintf("%s\t%x", p, sha256.Sum256(b))
			if c, err := LoadCase(p); err == nil && len(c.Pre.Constituents) > 0 {
				cfg := fixtureConfig(c, FinalisedRules)
				res, qerr := Quote(cfg, marketOfF(c), inputOf(c))
				cres, cerr := QuoteConservative(cfg, marketOfF(c), inputOf(c))
				line += "\t" + resultString(res, qerr) + "\t" + resultString(cres, cerr)
			}
			lines = append(lines, line)
			_, _ = h.Write([]byte(line + "\n"))
			n++
		}
		t.Logf("suite %-8s %3d files digest %x", suite, n, h.Sum(nil)[:8])
		require.Equal(t, map[string]int{"fixtures": 85, "a5": 43, "mq": 14, "parity": 2}[suite], n, suite)
	}
	if out := os.Getenv("LOTFLOW_DIGEST_OUT"); out != "" {
		require.NoError(t, os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644))
	}
}

func resultString(r *SwapResult, err error) string {
	if err != nil {
		return "err " + err.Error()
	}
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("out %s inexact %v", r.AmountOut, r.Inexact)
}
