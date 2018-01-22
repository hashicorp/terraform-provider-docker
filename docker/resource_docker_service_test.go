package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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
				resource "docker_network" "test_network" {
					name   = "testNetwork"
					driver = "overlay"
				}

				resource "docker_volume" "foo" {
					name = "testVolume"
				}

				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}
				
				resource "docker_secret" "service_secret" {
					name = "mysecret"
					data = "ewogICJrZXkiOiAiUVdFUlRZIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					hostname = "myfooservice"

					networks = ["${docker_network.test_network.name}"]
					network_mode = "vip"

					host {
						host = "testhost"
						ip = "10.0.1.0"
					}

					mounts = [
						{
							source = "${docker_volume.foo.name}"
							target = "/mount/test"
							type   = "volume"
							consistency = "default"
							read_only = true
							volume_labels {
								env = "dev"
								terraform = "true"
							}
						}
					]

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
							file_name   = "/configs.json"
						},
					]
				
					secrets = [
						{
							secret_id   = "${docker_secret.service_secret.id}"
							secret_name = "${docker_secret.service_secret.name}"
							file_name   = "/secrets.json"
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
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "5s"
						timeout  = "2s"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "hostname", "myfooservice"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.3251854989", "testNetwork"),
					resource.TestCheckResourceAttr("docker_service.foo", "network_mode", "vip"),
					resource.TestCheckResourceAttr("docker_service.foo", "host.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "host.1878413705.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "host.1878413705.ip", "10.0.1.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.bind_propagation", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.consistency", "default"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.read_only", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.source", "testVolume"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.target", "/mount/test"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.tmpfs_mode", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.type", "volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_driver_name", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_driver_options.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_labels.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_labels.env", "dev"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_labels.terraform", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.3510941185.volume_no_copy", "false"),
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
					// resource.TestCheckResourceAttr("docker_service.foo", "configs.1255247167.file_name", "/configs.json"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					// resource.TestCheckResourceAttr("docker_service.foo", "secrets.3229549426.config_name", "mysecret"),
					// resource.TestCheckResourceAttr("docker_service.foo", "secrets.3229549426.file_name", "/secrets.json"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.driver_name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-size", "10m"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-file", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
		},
	})
}

func TestAccDockerService_updateHealthcheck(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig-${uuid()}"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "2s"
						timeout  = "800ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
		},
	})
}
func TestAccDockerService_updateIncreaseReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 1
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 3
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
		},
	})
}
func TestAccDockerService_updateDecreaseReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 5
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 1
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
		},
	})
}

func TestAccDockerService_updateImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 2

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080/newroute", "new")),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "567")),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigAndSecret(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_secret" "service_secret" {
					name 			 = "mysecret-${replace(timestamp(),":", ".")}"
					data 			 = "ewogICJrZXkiOiAiUVdFUlRZIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					secrets = [
						{
							secret_id   = "${docker_secret.service_secret.id}"
							secret_name = "${docker_secret.service_secret.name}"
							file_name   = "/secrets.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name       = "myconfig-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_secret" "service_secret" {
					name       = "mysecret-${replace(timestamp(),":", ".")}"
					data       = "ewogICJrZXkiOiAiUVdFUlRZIgp9" # UPDATED to YXCVB
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					secrets = [
						{
							secret_id   = "${docker_secret.service_secret.id}"
							secret_name = "${docker_secret.service_secret.name}"
							file_name   = "/secrets.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "567")),
				),
			},
		},
	})
}

func TestAccDockerService_updateMultipleConfigs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name       = "myconfig-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}
				
				resource "docker_config" "service_config_2" {
					name       = "myconfig-2-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
						{
							config_id   = "${docker_config.service_config_2.id}"
							config_name = "${docker_config.service_config_2.name}"
							file_name   = "/configs2.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name       = "myconfig-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_config" "service_config_2" {
					name       = "myconfig-2-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
						{
							config_id   = "${docker_config.service_config_2.id}"
							config_name = "${docker_config.service_config_2.name}"
							file_name   = "/configs2.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "567")),
				),
			},
		},
	})
}

func TestAccDockerService_updatePort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8081"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 4

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports = [
						{
							internal = "8080"
							external = "8081"
						},
						{
							internal = "8080"
							external = "8082"
						}
					] 

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "123")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082", "123")),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigReplicasImageAndHealth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8081"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 4

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports = [
						{
							internal = "8080"
							external = "8081"
						},
						{
							internal = "8080"
							external = "8082"
						}
					] 

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "2s"
						timeout  = "800ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081/newroute", "new")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082/newroute", "new")),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigForMultipleServices(t *testing.T) {
	t.Skip("Skipping for travis ATM")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}

				resource "docker_service" "bar" {
					name     = "service-bar"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 5
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8085"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
					resource.TestMatchResourceAttr("docker_service.bar", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.bar", "name", "service-bar"),
					resource.TestCheckResourceAttr("docker_service.bar", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.bar", "replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.965731645.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.965731645.external", "8085"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8085", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				resource "docker_service" "bar" {
					name     = "service-bar"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 5
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8085"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "567")),
					resource.TestMatchResourceAttr("docker_service.bar", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.bar", "name", "service-bar"),
					resource.TestCheckResourceAttr("docker_service.bar", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.bar", "replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.bar", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.965731645.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.bar", "ports.965731645.external", "8085"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.bar", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8085", "567")),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigAndDecreaseReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 5
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 1
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8080"
					}
					
					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.4021806484.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8080", "567")),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigReplicasImageAndHealthIncreaseAndDecreaseReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v1"
					replicas = 2

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports {
						internal = "8080"
						external = "8081"
					}

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "1s"
						timeout  = "500ms"
						retries  = 2
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "123")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 6

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports = [
						{
							internal = "8080"
							external = "8081"
						},
						{
							internal = "8080"
							external = "8082"
						}
					] 

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "2s"
						timeout  = "800ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "6"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081/newroute", "new")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082/newroute", "new")),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9"
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}

				resource "docker_service" "foo" {
					name     = "service-foo"
					image    = "127.0.0.1:5000/my-private-service:v2"
					replicas = 3

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = 0.1
						order             = "start-first"
					}

					configs = [
						{
							config_id   = "${docker_config.service_config.id}"
							config_name = "${docker_config.service_config.name}"
							file_name   = "/configs.json"
						},
					]

					ports = [
						{
							internal = "8080"
							external = "8081"
						},
						{
							internal = "8080"
							external = "8082"
						}
					] 

					healthcheck {
						test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
						interval = "2s"
						timeout  = "800ms"
						retries  = 4
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "service-foo"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:5000/my-private-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1648302198.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.802625553.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082", "567")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8081/newroute", "new")),
					resource.ComposeTestCheckFunc(testCheckDockerServiceHasPrefix("http://localhost:8082/newroute", "new")),
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

/////////////
// Helpers
/////////////
func testCheckDockerServiceHasPrefix(url string, prefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Could not query")
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Could not read body")
		}

		bodyString := string(body[:])
		if !strings.HasPrefix(bodyString, prefix) {
			return fmt.Errorf("Reponse body '%s' for url %s does not have the prefix '%s'", bodyString, url, prefix)
		}

		return nil
	}
}
