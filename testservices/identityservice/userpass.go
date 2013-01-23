package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Implement the v2 User Pass form of identity (Keystone)

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Title   string `json:"title"`
}

type ErrorWrapper struct {
	Error ErrorResponse `json:"error"`
}

type UserPassRequest struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
		TenantName string `json:"tenantName"`
	} `json:"auth"`
}

type Endpoint struct {
	AdminURL    string `json:"adminURL"`
	InternalURL string `json:"internalURL"`
	PublicURL   string `json:"publicURL"`
	Region      string `json:"region"`
}

type Service struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Endpoints []Endpoint
}

type TokenResponse struct {
	Expires string `json:"expires"` // should this be a date object?
	Id      string `json:"id"`      // Actual token string
	Tenant  struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

type RoleResponse struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TenantId string `json:"tenantId"`
}

type UserResponse struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Roles []RoleResponse `json:"roles"`
}

type AccessResponse struct {
	Access struct {
		ServiceCatalog []Service     `json:"serviceCatalog"`
		Token          TokenResponse `json:"token"`
		User           UserResponse  `json:"user"`
	} `json:"access"`
}

// Taken from: http://docs.openstack.org/api/quick-start/content/index.html#Getting-Credentials-a00665
var exampleResponse = `{
    "access": {
        "serviceCatalog": [
            {
                "endpoints": [
                    {
                        "adminURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "internalURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "publicURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "nova", 
                "type": "compute"
            },
            {
                "endpoints": [
                    {
                        "adminURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "internalURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "publicURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "glance", 
                "type": "image"
            }, 
            {
                "endpoints": [
                    {
                        "adminURL": "https://nova-api.trystack.org:5443/v2.0", 
                        "internalURL": "https://keystone.trystack.org:5000/v2.0", 
                        "publicURL": "https://keystone.trystack.org:5000/v2.0", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "keystone", 
                "type": "identity"
            }
        ], 
        "token": {
            "expires": "2012-02-15T19:32:21", 
            "id": "5df9d45d-d198-4222-9b4c-7a280aa35666", 
            "tenant": {
                "id": "1", 
                "name": "admin"
            }
        }, 
        "user": {
            "id": "14", 
            "name": "annegentle", 
            "roles": [
                {
                    "id": "2", 
                    "name": "Member", 
                    "tenantId": "1"
                }
            ]
        }
    }
}`

type UserPass struct {
	Users
	services []Service
}

func NewUserPass() *UserPass {
	userpass := &UserPass{
		services: make([]Service, 0),
	}
	userpass.users = make(map[string]UserInfo)
	return userpass
}

func (u *UserPass) RegisterServiceProvider(name, serviceType string, serviceProvider ServiceProvider) {
	service := Service{name, serviceType, serviceProvider.Endpoints()}
	u.addService(service)
}

func (u *UserPass) addService(service Service) {
	u.services = append(u.services, service)
}

var internalError = []byte(`{
    "error": {
        "message": "Internal failure",
	"code": 500,
	"title": Internal Server Error"
    }
}`)

func (u *UserPass) ReturnFailure(w http.ResponseWriter, status int, message string) {
	e := ErrorWrapper{
		Error: ErrorResponse{
			Message: message,
			Code:    status,
			Title:   http.StatusText(status),
		},
	}
	if content, err := json.Marshal(e); err != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(internalError)))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(internalError)
	} else {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(status)
		w.Write(content)
	}
}

// Taken from an actual responses, however it may vary based on actual Openstack implementation
const (
	notJSON = ("Expecting to find application/json in Content-Type header." +
		" The server could not comply with the request since it is either malformed" +
		" or otherwise incorrect. The client is assumed to be in error.")
)

func (u *UserPass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req UserPassRequest
	// Testing against Canonistack, all responses are application/json, even failures
	w.Header().Set("Content-Type", "application/json")
	if r.Header.Get("Content-Type") != "application/json" {
		u.ReturnFailure(w, http.StatusBadRequest, notJSON)
		return
	}
	if content, err := ioutil.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		if err := json.Unmarshal(content, &req); err != nil {
			u.ReturnFailure(w, http.StatusBadRequest, notJSON)
			return
		}
	}
	userInfo, errmsg := u.authenticate(req.Auth.PasswordCredentials.Username, req.Auth.PasswordCredentials.Password)
	if errmsg != "" {
		u.ReturnFailure(w, http.StatusUnauthorized, errmsg)
		return
	}
	res := AccessResponse{}
	// We pre-populate the response with genuine entries so that it looks sane.
	// XXX: We should really build up valid state for this instead, at the
	//	very least, we should manage the URLs better.
	if err := json.Unmarshal([]byte(exampleResponse), &res); err != nil {
		u.ReturnFailure(w, http.StatusInternalServerError, err.Error())
		return
	}
	res.Access.ServiceCatalog = u.services
	res.Access.Token.Id = userInfo.Token
	res.Access.Token.Tenant.Id = userInfo.TenantId
	res.Access.User.Id = userInfo.Id
	if content, err := json.Marshal(res); err != nil {
		u.ReturnFailure(w, http.StatusInternalServerError, err.Error())
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
		return
	}
	panic("All paths should have already returned")
}

// setupHTTP attaches all the needed handlers to provide the HTTP API.
func (u *UserPass) SetupHTTP(mux *http.ServeMux) {
	mux.Handle("/tokens", u)
}
