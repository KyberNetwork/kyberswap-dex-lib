const getDataPath = (chain) => `../../internal/pkg/data/chain/${chain}/tokens.json`

export const collectData = () => {
    return {
        aurora: JSON.parse(open(getDataPath("aurora")).toString()).slice(0,6),
        avalanche: JSON.parse(open(getDataPath("avalanche")).toString()).slice(0,6),
        bsc: JSON.parse(open(getDataPath("bsc")).toString()).slice(0,6),
        cronos: JSON.parse(open(getDataPath("cronos")).toString()).slice(0,6),
        ethereum: JSON.parse(open(getDataPath("ethereum")).toString()).slice(0,6),
        fantom: JSON.parse(open(getDataPath("fantom")).toString()).slice(0,6),
        polygon: JSON.parse(open(getDataPath("polygon")).toString()).slice(0,6)
    }
}