package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	fw "github.com/mavogel/sshforward"
)

// Workaround for mutiple calls for configureFunc
// and no teardown method in provider https://github.com/hashicorp/terraform/issues/6258
var isForwardEstablished = false

// Provider creates the Docker provider
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:///var/run/docker.sock"),
				Description: "The Docker daemon address",
			},

			"ca_material": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CA_MATERIAL", ""),
				Description: "PEM-encoded content of Docker host CA certificate",
			},
			"cert_material": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_MATERIAL", ""),
				Description: "PEM-encoded content of Docker client certificate",
			},
			"key_material": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_KEY_MATERIAL", ""),
				Description: "PEM-encoded content of Docker client private key",
			},

			"cert_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_PATH", ""),
				Description: "Path to directory with Docker TLS config",
			},

			"registry_auth": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Address of the registry",
						},

						"username": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"registry_auth.config_file"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
							Description:   "Username for the registry",
						},

						"password": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							Sensitive:     true,
							ConflictsWith: []string{"registry_auth.config_file"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Description:   "Password for the registry",
						},

						"config_file": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"registry_auth.username", "registry_auth.password"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_CONFIG", "~/.docker/config.json"),
							Description:   "Path to docker json file for registry auth",
						},
					},
				},
			},

			"forward_config": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Configuration to forward the docker daemon from a remote to a local address",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bastion_host": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "The host address of the bastion host",
							ValidateFunc: validateStringMatchesPattern(`^.+:\d+$`),
						},
						"bastion_host_user": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The user to login via ssh on the bastion host",
						},
						"bastion_host_password": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							Sensitive:     true,
							ConflictsWith: []string{"forward_config.bastion_host_private_key_file"},
							Description:   "The password of the user to login via ssh on the bastion host",
						},
						"bastion_host_private_key_file": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"forward_config.bastion_host_password"},
							Description:   "The private key file associated with the user to login via ssh on the bastion host",
						},
						"end_host": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The host address of the end host",
							ValidateFunc: validateStringMatchesPattern(`^.+:\d+$`),
						},
						"end_host_user": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The user to login via ssh on the end host",
						},
						"end_host_password": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							Sensitive:     true,
							ConflictsWith: []string{"forward_config.end_host_private_key_file"},
							Description:   "The password of the user to login via ssh on the end host",
						},
						"end_host_private_key_file": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"forward_config.end_host_password"},
							Description:   "The private key file associated with the user to login via ssh on the end host",
						},
						"local_address": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The local address the docker daemon is forwarded to",
							ValidateFunc: validateStringMatchesPattern(`^.+:\d+$`),
						},
						"remote_address": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The address on the remote/end host the docker daemon is forwarded from",
							ValidateFunc: validateStringMatchesPattern(`^.+:\d+$`),
						},
					},
				},
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"docker_container": resourceDockerContainer(),
			"docker_image":     resourceDockerImage(),
			"docker_network":   resourceDockerNetwork(),
			"docker_volume":    resourceDockerVolume(),
			"docker_config":    resourceDockerConfig(),
			"docker_secret":    resourceDockerSecret(),
			"docker_service":   resourceDockerService(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"docker_registry_image": dataSourceDockerRegistryImage(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Host:     d.Get("host").(string),
		Ca:       d.Get("ca_material").(string),
		Cert:     d.Get("cert_material").(string),
		Key:      d.Get("key_material").(string),
		CertPath: d.Get("cert_path").(string),
	}

	if forwardConfig, ok := d.GetOk("forward_config"); ok {
		if !isForwardEstablished {
			if err := createForward(forwardConfig.([]interface{})); err != nil {
				return nil, fmt.Errorf("Error creating forward: %s", err)
			}
			isForwardEstablished = true
		}
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, fmt.Errorf("Error initializing Docker client: %s", err)
	}

	ctx := context.Background()
	_, err = client.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error pinging Docker server: %s", err)
	}

	authConfigs := &AuthConfigs{}

	if v, ok := d.GetOk("registry_auth"); ok {
		authConfigs, err = providerSetToRegistryAuth(v.(*schema.Set))

		if err != nil {
			return nil, fmt.Errorf("Error loading registry auth config: %s", err)
		}
	}

	providerConfig := ProviderConfig{
		DockerClient: client,
		AuthConfigs:  authConfigs,
	}

	return &providerConfig, nil
}

// ErrCannotParseDockercfg is the error returned by NewAuthConfigurations when the dockercfg cannot be parsed.
var ErrCannotParseDockercfg = errors.New("Failed to read authentication from dockercfg")

// AuthConfigs represents authentication options to use for the
// PushImage method accommodating the new X-Registry-Config header
type AuthConfigs struct {
	Configs map[string]types.AuthConfig `json:"configs"`
}

// dockerConfig represents a registry authentation configuration from the
// .dockercfg file.
type dockerConfig struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

// Take the given registry_auth schemas and return a map of registry auth configurations
func providerSetToRegistryAuth(authSet *schema.Set) (*AuthConfigs, error) {
	authConfigs := AuthConfigs{
		Configs: make(map[string]types.AuthConfig),
	}

	for _, authInt := range authSet.List() {
		auth := authInt.(map[string]interface{})
		authConfig := types.AuthConfig{}
		authConfig.ServerAddress = normalizeRegistryAddress(auth["address"].(string))

		// For each registry_auth block, generate an AuthConfiguration using either
		// username/password or the given config file
		if username, ok := auth["username"]; ok && username.(string) != "" {
			authConfig.Username = auth["username"].(string)
			authConfig.Password = auth["password"].(string)
		} else if configFile, ok := auth["config_file"]; ok && configFile.(string) != "" {
			filePath := configFile.(string)
			if strings.HasPrefix(filePath, "~/") {
				usr, err := user.Current()
				if err != nil {
					return nil, err
				}
				filePath = strings.Replace(filePath, "~", usr.HomeDir, 1)
			}

			r, err := os.Open(filePath)
			if err != nil {
				return nil, fmt.Errorf("Error opening docker registry config file: %v", err)
			}

			auths, err := newAuthConfigurations(r)
			if err != nil {
				return nil, fmt.Errorf("Error parsing docker registry config json: %v", err)
			}

			foundRegistry := false
			for registry, authFileConfig := range auths.Configs {
				if authConfig.ServerAddress == normalizeRegistryAddress(registry) {
					authConfig.Username = authFileConfig.Username
					authConfig.Password = authFileConfig.Password
					foundRegistry = true
				}
			}

			if !foundRegistry {
				return nil, fmt.Errorf("Couldn't find registry config for '%s' in file: %s",
					authConfig.ServerAddress, filePath)
			}
		}

		authConfigs.Configs[authConfig.ServerAddress] = authConfig
	}

	return &authConfigs, nil
}

// newAuthConfigurations returns AuthConfigs from a JSON encoded string in the
// same format as the .dockercfg file.
func newAuthConfigurations(r io.Reader) (*AuthConfigs, error) {
	var auth *AuthConfigs
	confs, err := parseDockerConfig(r)
	if err != nil {
		return nil, err
	}
	auth, err = authConfigs(confs)
	if err != nil {
		return nil, err
	}
	return auth, nil
}

// parseDockerConfig parses the docker config file for auths
func parseDockerConfig(r io.Reader) (map[string]dockerConfig, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	byteData := buf.Bytes()

	confsWrapper := struct {
		Auths map[string]dockerConfig `json:"auths"`
	}{}
	if err := json.Unmarshal(byteData, &confsWrapper); err == nil {
		if len(confsWrapper.Auths) > 0 {
			return confsWrapper.Auths, nil
		}
	}

	var confs map[string]dockerConfig
	if err := json.Unmarshal(byteData, &confs); err != nil {
		return nil, err
	}
	return confs, nil
}

// authConfigs converts a dockerConfigs map to a AuthConfigs object.
func authConfigs(confs map[string]dockerConfig) (*AuthConfigs, error) {
	c := &AuthConfigs{
		Configs: make(map[string]types.AuthConfig),
	}
	for reg, conf := range confs {
		if conf.Auth == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(conf.Auth)
		if err != nil {
			return nil, err
		}
		userpass := strings.SplitN(string(data), ":", 2)
		if len(userpass) != 2 {
			return nil, ErrCannotParseDockercfg
		}
		c.Configs[reg] = types.AuthConfig{
			Email:         conf.Email,
			Username:      userpass[0],
			Password:      userpass[1],
			ServerAddress: reg,
			Auth:          conf.Auth,
		}
	}
	return c, nil
}

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

func createForward(forwardConfig []interface{}) error {
	parsedForwardConfig, err := parseForwardConfig(forwardConfig)
	if err != nil {
		return fmt.Errorf("Invalid forward config: %s", err)
	}

	// NOTE: when teardown exists https://github.com/hashicorp/terraform/issues/6258
	// we should close the forward then
	_, _, bootstrapErr := fw.NewForward(parsedForwardConfig)
	if bootstrapErr != nil {
		return fmt.Errorf("Could not establish the forward: %s", bootstrapErr)
	}
	return nil
}
