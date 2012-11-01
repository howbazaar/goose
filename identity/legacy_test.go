package identity

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type LegacyTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&LegacyTestSuite{})

func (s *LegacyTestSuite) TestAuthAgainstServer(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	service.AddUser("joe-user", "secrets", "active-token")
	service.SetManagementURL("http://management/url")
	l := Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, "active-token")
	c.Assert(auth.ServiceURLs, DeepEquals, map[string]string{"compute": "http://management/url"})
}

func (s *LegacyTestSuite) TestBadAuth(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	service.AddUser("joe-user", "secrets", "active-token")
	l := Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "bad-secrets"}
	auth, err := l.Auth(creds)
	c.Assert(err, NotNil)
	c.Assert(auth, IsNil)
}
