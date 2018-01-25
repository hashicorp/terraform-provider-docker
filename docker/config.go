package docker

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	dc "github.com/fsouza/go-dockerclient"
	fw "github.com/mavogel/sshforward"
)

// DockerConfig is the structure that stores the configuration to talk to a
// Docker API compatible host.
type DockerConfig struct {
	Host          string
	Ca            string
	Cert          string
	Key           string
	CertPath      string
	ForwardConfig []interface{}
}

// NewClient() returns a new Docker client.
func (c *DockerConfig) NewClient() (*dc.Client, error) {
	if c.Ca != "" || c.Cert != "" || c.Key != "" {
		if c.Ca == "" || c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("ca_material, cert_material, and key_material must be specified")
		}

		if c.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		if c.ForwardConfig != nil {
			forwardConfig, err := parseForwardConfig(c.ForwardConfig)
			if err != nil {
				return nil, fmt.Errorf("Invalid forward config: %s", err)
			}
			_, _, bootstrapErr := fw.NewForward(forwardConfig)
			if bootstrapErr != nil {
				log.Printf("bootstrapErr: %s", bootstrapErr)
				return nil, fmt.Errorf("bootstrapErr while estblishing forward: %s", bootstrapErr)
			}
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

// Takes the given forward config and parses it into the fw.Config
func parseForwardConfig(forwardConfigList []interface{}) (*fw.Config, error) {
	forwardConfig := &fw.Config{}

	if forwardConfigList != nil && len(forwardConfigList) > 0 {
		fc := forwardConfigList[0].(map[string]interface{})

		if v, ok := fc["bastion_host"]; ok {
			jumpHostConfigs := make([]*fw.SSHConfig, 0)
			bastionHostConfig := &fw.SSHConfig{}
			bastionHostConfig.Address = v.(string)
			if v, ok := fc["bastion_host_user"]; ok {
				bastionHostConfig.User = v.(string)
			}
			if v, ok := fc["bastion_host_password"]; ok {
				bastionHostConfig.Password = v.(string)
			}
			if v, ok := fc["bastion_host_private_key_file"]; ok {
				bastionHostConfig.PrivateKeyFile = v.(string)
			}
			jumpHostConfigs = append(jumpHostConfigs, bastionHostConfig)
			forwardConfig.JumpHostConfigs = jumpHostConfigs
		}

		forwardConfig.EndHostConfig = &fw.SSHConfig{}
		if v, ok := fc["end_host"]; ok {
			forwardConfig.EndHostConfig.Address = v.(string)
		}
		if v, ok := fc["end_host_user"]; ok {
			forwardConfig.EndHostConfig.User = v.(string)
		}
		if v, ok := fc["end_host_password"]; ok {
			forwardConfig.EndHostConfig.Password = v.(string)
		}
		if v, ok := fc["end_host_private_key_file"]; ok {
			forwardConfig.EndHostConfig.PrivateKeyFile = v.(string)
		}

		if v, ok := fc["local_address"]; ok {
			forwardConfig.LocalAddress = v.(string)
		}

		if v, ok := fc["remote_address"]; ok {
			forwardConfig.RemoteAddress = v.(string)
		}
	}

	return forwardConfig, nil
}
