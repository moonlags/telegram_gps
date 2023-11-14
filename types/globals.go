package types

import (
	"os"

	"golang.org/x/oauth2"
)

var githubConf *oauth2.Config

func GithubConf() *oauth2.Config {
	return githubConf
}

func InitGithubConf() {
	githubConf = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_SECRET"),
		RedirectURL:  "http://localhost:55081/callback/github",
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
}
