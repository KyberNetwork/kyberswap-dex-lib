package abis

import _ "embed"

var (
	//go:embed Core.json
	coreJson []byte

	//go:embed Twamm.json
	twammJson []byte

	//go:embed QuoteDataFetcher.json
	quoteDataFetcherJson []byte

	//go:embed TwammDataFetcher.json
	twammDataFetcherJson []byte

	//go:embed MevCaptureRouter.json
	mevCaptureRouterJson []byte
)
