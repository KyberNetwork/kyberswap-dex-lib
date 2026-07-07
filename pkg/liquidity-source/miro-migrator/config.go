package miromigrator

type Config struct {
	Migrator string `json:"migrator"`
	PSP      string `json:"psp"`
	SePSP1   string `json:"sepsp1"`
	VLR      string `json:"vlr"`
}
