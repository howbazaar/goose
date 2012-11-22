package identity

import (
	"fmt"
	"os"
)

type AuthMethod int

const (
	AUTH_LEGACY = AuthMethod(iota)
	AUTH_USERPASS
)

func (a AuthMethod) String() string {
	switch a {
	case AUTH_LEGACY:
		return "Legacy Authenetication"
	case AUTH_USERPASS:
		return "Username/password Authentication"
	}
	panic(fmt.Errorf("Unknown athentication type: %d", a))
}

type AuthDetails struct {
	Token       string
	TenantId    string
	UserId      string
	ServiceURLs map[string]string
}

type Credentials struct {
	URL        string // The URL to authenticate against
	User       string // The username to authenticate as
	Secrets    string // The secrets to pass
	Region     string // Region to send requests to
	TenantName string // The tenant information for this connection
}

type Authenticator interface {
	Auth(creds *Credentials) (*AuthDetails, error)
}

func getConfig(envVars ...string) (value string) {
	value = ""
	for _, v := range envVars {
		value = os.Getenv(v)
		if value != "" {
			break
		}
	}
	return
}

func CredentialsFromEnv() *Credentials {
	return &Credentials{
		URL:        getConfig("OS_AUTH_URL"),
		User:       getConfig("OS_USERNAME", "NOVA_USERNAME"),
		Secrets:    getConfig("OS_PASSWORD", "NOVA_PASSWORD"),
		Region:     getConfig("OS_REGION_NAME", "NOVA_REGION"),
		TenantName: getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID"),
	}
}
