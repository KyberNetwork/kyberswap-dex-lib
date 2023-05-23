package source

type EnabledDexes []string

type ScanDex struct {
	Id         string                 `mapstructure:"id"`
	Handler    string                 `mapstructure:"handler"`
	Json       bool                   `mapstructure:"json"`
	Properties map[string]interface{} `mapstructure:"properties"`
}
