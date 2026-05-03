package models

type OauthResult struct {
	AppScheme        string
	Accesstoken      string
	RefreshToken     string
	RedirectDeepLink string
	Userid           string
}

type GithubUserInfo struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	HTMLURL     string `json:"html_url"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	CreatedAt   string `json:"created_at"`
}

type GoogleUserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	AvatarURL string `json:"picture"`
	Name      string `json:"name"`
}

type Oauth struct{}
