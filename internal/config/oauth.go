package config

type OAuthConfig struct {
	AppScheme      string `mapstructure:"app_scheme"`
	MobileRedirect string `mapstructure:"mobile_redirect"`

	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	GoogleRedirectURL  string `mapstructure:"google_redirect_url"`

	GithubClientID     string `mapstructure:"github_client_id"`
	GithubClientSecret string `mapstructure:"github_client_secret"`
	GithubRedirectURL  string `mapstructure:"github_redirect_url"`
}
