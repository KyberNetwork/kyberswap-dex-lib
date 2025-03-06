package kafka

type Config struct {
	Addresses         []string `mapstructure:"addresses" json:"addresses" default:""`
	UseAuthentication bool     `mapstructure:"useAuthentication" json:"useAuthentication" default:""`
	Username          string   `mapstructure:"username" json:"username" default:""`
	Password          string   `mapstructure:"password" json:"-" default:""`
	Enable            bool     `mapstructure:"enable" json:"-" default:""`
}
