package docker

import (
	"fmt"
	"path/filepath"
	"strings"

	dc "github.com/fsouza/go-dockerclient"
)

// DockerConfig is the structure that stores the configuration to talk to a
// Docker API compatible host.
type DockerConfig struct {
	Host     string
	Ca       string
	Cert     string
	Key      string
	CertPath string
}

// NewClient returns a new Docker client.
func (c *DockerConfig) NewClient() (*dc.Client, error) {
	if c.Ca != "" || c.Cert != "" || c.Key != "" {
		if c.Ca == "" || c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("ca_material, cert_material, and key_material must be specified")
		}

		if c.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		return dc.NewTLSClientFromBytes(c.Host, []byte(c.Cert), []byte(c.Key), []byte(c.Ca))
	}

	if c.CertPath != "" {
		// If there is cert information, load it and use it.
		ca := filepath.Join(c.CertPath, "ca.pem")
		cert := filepath.Join(c.CertPath, "cert.pem")
		key := filepath.Join(c.CertPath, "key.pem")

		return dc.NewTLSClient(c.Host, cert, key, ca)
	}

	// If there is no cert information, then just return the direct client
	return dc.NewClient(c.Host)
}

// Data structure for holding data that we fetch from Docker.
type Data struct {
	DockerImages map[string]*dc.APIImages
}

// ProviderConfig for the custom registry provider
type ProviderConfig struct {
	DockerClient *dc.Client
	AuthConfigs  *dc.AuthConfigurations
}

// The registry address can be referenced in various places (registry auth, docker config file, image name)
// with or without the http(s):// prefix; this function is used to standardize the inputs
func normalizeRegistryAddress(address string) string {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}
