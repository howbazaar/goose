// Swift double testing service - internal direct API implementation

package swiftservice

import (
	"fmt"
	"launchpad.net/goose/swift"
	"launchpad.net/goose/testservices/identityservice"
	"net/url"
	"strings"
	"time"
)

type object map[string][]byte

type Swift struct {
	identityservice.ServiceInstance
	containers map[string]object
}

// New creates an instance of the Swift object, given the parameters.
func New(hostURL, versionPath, tenantId, region string, identityService identityservice.IdentityService) *Swift {
	url, err := url.Parse(hostURL)
	if err != nil {
		panic(err)
	}
	hostname := url.Host
	if !strings.HasSuffix(hostname, "/") {
		hostname += "/"
	}
	swift := &Swift{
		containers: make(map[string]object),
		ServiceInstance: identityservice.ServiceInstance{
			IdentityService: identityService,
			Hostname:        hostname,
			VersionPath:     versionPath,
			TenantId:        tenantId,
			Region:          region,
		},
	}
	if identityService != nil {
		identityService.RegisterServiceProvider("swift", "object-store", swift)
	}
	return swift
}

func (s *Swift) endpointURL(path string) string {
	ep := "http://" + s.Hostname + s.VersionPath + "/" + s.TenantId
	if path != "" {
		ep += "/" + strings.TrimLeft(path, "/")
	}
	return ep
}

func (s *Swift) Endpoints() []identityservice.Endpoint {
	ep := identityservice.Endpoint{
		AdminURL:    s.endpointURL(""),
		InternalURL: s.endpointURL(""),
		PublicURL:   s.endpointURL(""),
		Region:      s.Region,
	}
	return []identityservice.Endpoint{ep}
}

// HasContainer verifies the given container exists or not.
func (s *Swift) HasContainer(name string) bool {
	_, ok := s.containers[name]
	return ok
}

// GetObject retrieves a given object from its container, returning
// the object data or an error.
func (s *Swift) GetObject(container, name string) ([]byte, error) {
	data, ok := s.containers[container][name]
	if !ok {
		return nil, fmt.Errorf("no such object %q in container %q", name, container)
	}
	return data, nil
}

// AddContainer creates a new container with the given name, if it
// does not exist. Otherwise an error is returned.
func (s *Swift) AddContainer(name string) error {
	if s.HasContainer(name) {
		return fmt.Errorf("container already exists %q", name)
	}
	s.containers[name] = make(object)
	return nil
}

// ListContainer lists the objects in the given container.
func (s *Swift) ListContainer(name string) ([]swift.ContainerContents, error) {
	if ok := s.HasContainer(name); !ok {
		return nil, fmt.Errorf("no such container %q", name)
	}
	items := s.containers[name]
	contents := make([]swift.ContainerContents, len(items))
	var i = 0
	for k, v := range items {
		contents[i] = swift.ContainerContents{
			Name:         k,
			Hash:         "", // not implemented
			LengthBytes:  len(v),
			ContentType:  "application/octet-stream",
			LastModified: time.Now().Format("2006-01-02 15:04:05"), //not implemented
		}
		i++
	}
	return contents, nil
}

// AddObject creates a new object with the given name in the specified
// container, setting the object's data. It's an error if the object
// already exists. If the container does not exist, it will be
// created.
func (s *Swift) AddObject(container, name string, data []byte) error {
	if _, err := s.GetObject(container, name); err == nil {
		return fmt.Errorf(
			"object %q in container %q already exists",
			name,
			container)
	}
	if ok := s.HasContainer(container); !ok {
		if err := s.AddContainer(container); err != nil {
			return err
		}
	}
	s.containers[container][name] = data
	return nil
}

// RemoveContainer deletes an existing container with the given name.
func (s *Swift) RemoveContainer(name string) error {
	if ok := s.HasContainer(name); !ok {
		return fmt.Errorf("no such container %q", name)
	}
	delete(s.containers, name)
	return nil
}

// RemoveObject deletes an existing object in a given container.
func (s *Swift) RemoveObject(container, name string) error {
	if _, err := s.GetObject(container, name); err != nil {
		return err
	}
	delete(s.containers[container], name)
	return nil
}

// GetURL returns the full URL, which can be used to GET the
// object. An error occurs if the object does not exist.
func (s *Swift) GetURL(container, object string) (string, error) {
	if _, err := s.GetObject(container, object); err != nil {
		return "", err
	}
	return s.endpointURL(fmt.Sprintf("%s/%s", container, object)), nil
}
