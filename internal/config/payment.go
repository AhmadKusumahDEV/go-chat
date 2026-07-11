package config

type PaymentConfig struct {
	Midtrans Midtrans `mapstructure:"midtrans"`
}

type Midtrans struct {
	ClientKey             string   `mapstructure:"client_key"`
	ServerKey             string   `mapstructure:"server_key"`
	MerchantId            string   `mapstructure:"merchant_id"`
	Mode                  string   `mapstructure:"mode"`
	UrlSnapSanbox         string   `mapstructure:"url_snap_sanbox"`
	UrlSnapProduction     string   `mapstructure:"url_snap_production"`
	UrlRedirectSanbox     string   `mapstructure:"url_redirect_sandbox"`
	UrlRedirectProduction string   `mapstructure:"url_redirect_production"`
	AllowMethodPayment    []string `mapstructure:"allow_payment_methods"`
}
