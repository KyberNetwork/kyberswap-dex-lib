// This is a nested module boundary, not a real Go module: pure test-fixture data, no .go files.
// Go's module-zip packer excludes any subdirectory containing its own go.mod, so this directory
// is skipped entirely when this repo is fetched as a dependency (go get/go mod download) by
// downstream services -- only a full git clone of this repo still includes it. See AGENTS.md
// "Test data size".
module github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/everlong/flamm/testdata

go 1.25.10
