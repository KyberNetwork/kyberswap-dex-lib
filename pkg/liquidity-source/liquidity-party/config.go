package liquidityparty

// Config follows the Kyber convention of a per-DEX config populated from external Properties JSON
// (the euler-swap shared.Config pattern). DexID is auto-injected by the list/tracker factories from
// the exchange constant; the PartyPlanner (factory + on-chain pool index) and PartyInfo (view/quote
// singleton) addresses come from Kyber's per-chain config rather than being committed as Go constants.
type Config struct {
	DexID               string `json:"dexID"`
	PartyPlannerAddress string `json:"partyPlannerAddress"`
	PartyInfoAddress    string `json:"partyInfoAddress"`
}
