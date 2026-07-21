package abis

import _ "embed"

var (
	//go:embed Core.json
	coreJson []byte

	//go:embed TWAMM.json
	twammJson []byte

	//go:embed QuoteDataFetcher.json
	quoteDataFetcherJson []byte

	//go:embed TWAMMDataFetcher.json
	twammDataFetcherJson []byte

	//go:embed MEVCaptureRouter.json
	mevCaptureRouterJson []byte

	//go:embed BoostedFees.json
	boostedFeesJson []byte

	//go:embed BoostedFeesDataFetcher.json
	boostedFeesDataFetcherJson []byte

	//go:embed Ve33.json
	ve33Json []byte

	//go:embed Ve33DataFetcher.json
	ve33DataFetcherJson []byte
)
