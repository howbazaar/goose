package client_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/openstack"
)

func registerLocalTests(authMethods []identity.AuthMethod) {
	for _, authMethod := range authMethods {
		Suite(&localLiveSuite{
			LiveTests: LiveTests{
				authMethod: authMethod,
			},
		})
	}
}

// localLiveSuite runs tests from LiveTests using a fake
// identity server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	service identityservice.HttpService
}

func (s *localLiveSuite) SetUpSuite(c *C) {
	c.Logf("Using identity service test double")
	s.HTTPSuite.SetUpSuite(c)
	s.cred = &identity.Credentials{
		URL:     s.Server.URL,
		User:    "fred",
		Secrets: "secret",
		Region:  "some region"}
	switch s.authMethod {
	default:
		panic("Invalid authentication method")
	case identity.AuthUserPass:
		// The openstack test service sets up userpass authentication.
		s.service = openstack.New(s.Server.URL, s.cred.User, s.cred.Secrets, s.LiveTests.cred.Region)
	case identity.AuthLegacy:
		legacy := identityservice.NewLegacy()
		legacy.AddUser(s.cred.User, s.cred.Secrets)
		legacy.SetManagementURL("http://management.test.invalid/url")
		s.service = legacy
	}
	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *C) {
	s.LiveTests.TearDownSuite(c)
	s.HTTPSuite.TearDownSuite(c)
}

func (s *localLiveSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
	s.LiveTests.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *C) {
	s.LiveTests.TearDownTest(c)
	s.HTTPSuite.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.
