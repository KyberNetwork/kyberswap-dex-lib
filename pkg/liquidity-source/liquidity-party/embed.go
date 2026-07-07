package liquidityparty

import _ "embed"

//go:embed abis/PartyPlanner.json
var partyPlannerABIJson []byte

//go:embed abis/PartyPool.json
var partyPoolABIJson []byte

//go:embed abis/PartyInfo.json
var partyInfoABIJson []byte
