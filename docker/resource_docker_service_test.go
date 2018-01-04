package docker

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
)

// ----------------------------------------
// -----------    UNIT  TESTS   -----------
// ----------------------------------------

func TestDockerSecretFromRegistryAuth_basic(t *testing.T) {
	authConfigs := make(map[string]dc.AuthConfiguration)
	authConfigs["https://repo.my-company.com:8787"] = dc.AuthConfiguration{
		Username:      "myuser",
		Password:      "mypass",
		Email:         "",
		ServerAddress: "repo.my-company.com:8787",
	}

	foundAuthConfig := fromRegistryAuth("repo.my-company.com:8787/my_image", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "myuser")
	checkAttribute(t, "Password", foundAuthConfig.Password, "mypass")
	checkAttribute(t, "Email", foundAuthConfig.Email, "")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "repo.my-company.com:8787")
}

func TestDockerSecretFromRegistryAuth_multiple(t *testing.T) {
	authConfigs := make(map[string]dc.AuthConfiguration)
	authConfigs["https://repo.my-company.com:8787"] = dc.AuthConfiguration{
		Username:      "myuser",
		Password:      "mypass",
		Email:         "",
		ServerAddress: "repo.my-company.com:8787",
	}
	authConfigs["https://nexus.my-fancy-company.com"] = dc.AuthConfiguration{
		Username:      "myuser33",
		Password:      "mypass123",
		Email:         "test@example.com",
		ServerAddress: "nexus.my-fancy-company.com",
	}

	foundAuthConfig := fromRegistryAuth("nexus.my-fancy-company.com/the_image", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "myuser33")
	checkAttribute(t, "Password", foundAuthConfig.Password, "mypass123")
	checkAttribute(t, "Email", foundAuthConfig.Email, "test@example.com")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "nexus.my-fancy-company.com")

	foundAuthConfig = fromRegistryAuth("alpine:3.1", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "")
	checkAttribute(t, "Password", foundAuthConfig.Password, "")
	checkAttribute(t, "Email", foundAuthConfig.Email, "")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "")
}

func checkAttribute(t *testing.T, name, actual, expected string) error {
	if actual != expected {
		t.Fatalf("bad authconfig attribute for '%q'\nExpected: %s\n     Got: %s", name, expected, actual)
	}

	return nil
}

// ----------------------------------------
// ----------- ACCEPTANCE TESTS -----------
// ----------------------------------------
var serviceIDRegex = regexp.MustCompile(`[A-Za-z0-9_\+\.-]+`)

func TestAccDockerService_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "stovogel/friendlyhello:part2"
					replicas = 2
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_full(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig-${replace(timestamp(),":", ".")}"
					data = "eyJhIjoiYiJ9"

					lifecycle {
						ignore_changes = ["name"]
					}
				}
				
				resource "docker_secret" "service_secret" {
					name = "mysecret-${replace(timestamp(),":", ".")}"
					data = "eyJhIjoiYiJ9"

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service"
					replicas = 2
					
					update_config {
						parallelism       = 2
						delay             = "10s"
						failure_action    = "pause"
						monitor           = "5s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}
					
					rollback_config {
						parallelism       = 2
						delay             = "5ms"
						failure_action    = "pause"
						monitor           = "10h"
						max_failure_ratio = "0.9"
						order             = "stop-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/root/configs.json"
						},
					]
				
					secrets = [
						{
							secret_id   = "${docker_secret.service_secret.id}"
							secret_name = "${docker_secret.service_secret.name}"
							file_name   = "/root/secrets.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					logging {
						driver_name = "json-file"
					
						options {
							max-size = "10m"
							max-file = "3"
						}
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080"]
						interval = "15s"
						timeout  = "10s"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.delay", "5ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.monitor", "10h"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.max_failure_ratio", "0.9"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					//  Note: the hash changes every time due to the usage of timestamp()
					// resource.TestCheckResourceAttr("docker_service.foo", "configs.1255247167.config_name", "myconfig"),
					// resource.TestCheckResourceAttr("docker_service.foo", "configs.1255247167.file_name", "/root/configs.json"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					// resource.TestCheckResourceAttr("docker_service.foo", "secrets.3229549426.config_name", "mysecret"),
					// resource.TestCheckResourceAttr("docker_service.foo", "secrets.3229549426.file_name", "/root/secrets.json"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1093694028.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1093694028.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.driver_name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-size", "10m"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-file", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "15s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

func TestAccDockerService_private(t *testing.T) {
	registry := os.Getenv("DOCKER_REGISTRY_ADDRESS")
	image := os.Getenv("DOCKER_PRIVATE_IMAGE")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
					provider "docker" {
						alias = "private"
						registry_auth {
							address = "%s"
						}
					}

					resource "docker_service" "bar" {
						provider = "docker.private"
						name     = "service-bar"
						image    = "%s"
						replicas = 2
					}
				`, registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.bar", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.bar", "name", "service-bar"),
					resource.TestCheckResourceAttr("docker_service.bar", "image", image),
					resource.TestCheckResourceAttr("docker_service.bar", "replicas", "2"),
				),
			},
		},
	})
}
