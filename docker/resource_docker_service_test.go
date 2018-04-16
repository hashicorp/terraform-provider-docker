package docker

import (
	"fmt"
	"log"
	"os"
	"regexp"
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
// Fire and Forget
var serviceIDRegex = regexp.MustCompile(`[A-Za-z0-9_\+\.-]+`)

func TestAccDockerService_plain(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					image    = "stovogel/friendlyhello:part2"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
				),
			},
		},
	})
}

func TestAccDockerService_basicReplicated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_basicGlobal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					image    = "stovogel/friendlyhello:part2"
					mode {
						global = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.global", "true"),
				),
			},
		},
	})
}

func TestAccDockerService_GlobalAndReplicated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
						global = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*conflicts with.*`),
			},
		},
	})
}
func TestAccDockerService_GlobalWithConvergeConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					image    = "stovogel/friendlyhello:part2"
					mode {
						global = true
					}
					converge_config {
						interval = "500ms"
						monitor  = "5s"
						timeout  = "20s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*conflicts with.*`),
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
					name = "tftest-fnf-up-image-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-image"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-fnf-up-image-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-image"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
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
					name 			 = "tftests-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 6
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3516784273.replicas", "6"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 3
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.2901027540.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

// Converging tests
func TestAccDockerService_nonExistingPrivateImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-privateimagedoesnotexist"
					image    = "127.0.0.1:15000/idonoexist:latest"
					mode {
						replicated {
							replicas = 2
						}
					}

					converge_config {
						interval = "500ms"
						monitor  = "5s"
						timeout  = "20s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not converge after.*`),
				Check: resource.ComposeTestCheckFunc(
					isServiceRemoved("tftest-service-privateimagedoesnotexist"),
				),
			},
		},
	})
}
func TestAccDockerService_nonExistingPublicImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-publicimagedoesnotexist"
					image    = "stovogel/blablabla:part5"
					mode {
						replicated {
							replicas = 2
						}
					}

					converge_config {
						interval = "500ms"
						monitor  = "5s"
						timeout  = "10s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not converge after.*`),
				Check: resource.ComposeTestCheckFunc(
					isServiceRemoved("tftest-service-publicimagedoesnotexist"),
				),
			},
		},
	})
}

func TestAccDockerService_fullConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_volume" "test_volume" {
					name = "tftest-volume"
				}

				resource "docker_config" "service_config" {
					name = "tftest-full-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}
				
				resource "docker_secret" "service_secret" {
					name = "tftest-mysecret"
					data = "ewogICJrZXkiOiAiUVdFUlRZIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-full"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					hostname = "myfooservice"

					networks = ["${docker_network.test_network.name}"]
					endpoint_mode = "vip"

					hosts {
						host = "testhost"
						ip = "10.0.1.0"
					}

					destroy_grace_seconds = "10"

					placement {
						constraints = [
							"node.role==manager"
						]
						prefs = [
							"spread=node.role.manager"
						]
					}

					mounts = [
						{
							source = "${docker_volume.test_volume.name}"
							target = "/mount/test"
							type   = "volume"
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
						max_failure_ratio = "0.1"
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
						internal 		 = "8080"
						external 		 = "8080"
						publish_mode = "ingress"
						protocol     = "tcp"
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

					dns_config {
						nameservers = ["8.8.8.8"]
						search = ["example.org"]
						options = ["timeout:3"]
					}
					
					converge_config {
						interval = "500ms"
						monitor  = "10s"
						timeout  = "1m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-full"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "hostname", "myfooservice"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.2768978924", "tftest-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_mode", "vip"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.0.constraints.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.0.constraints.4248571116", "node.role==manager"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.0.platforms.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.0.prefs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "placement.0.prefs.1751004438", "spread=node.role.manager"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.1878413705.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.1878413705.ip", "10.0.1.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "destroy_grace_seconds", "10"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.bind_propagation", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.read_only", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.source", "tftest-volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.target", "/mount/test"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.tmpfs_mode", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.type", "volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_driver_name", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_driver_options.%", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_labels.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_labels.env", "dev"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_labels.terraform", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.1197577087.volume_no_copy", "false"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.delay", "5ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.monitor", "10h"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.max_failure_ratio", "0.9"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.publish_mode", "ingress"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.protocol", "tcp"),
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
					resource.TestCheckResourceAttr("docker_service.foo", "dns_config.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "dns_config.0.nameservers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("docker_service.foo", "dns_config.0.search.0", "example.org"),
					resource.TestCheckResourceAttr("docker_service.foo", "dns_config.0.options.0", "timeout:3"),
				),
			},
		},
	})
}

func TestAccDockerService_updateFailsAndRollbackConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-rollback-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-rollback"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 4
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "5s"
						failure_action    = "rollback"
						monitor           = "10s"
						max_failure_ratio = "0.0"
						order             = "stop-first"
					}

					rollback_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "4s"
						max_failure_ratio = "0.0"
						order             = "stop-first"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
						timeout  = "3m"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-rollback"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3819682835.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "rollback"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.monitor", "4s"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.max_failure_ratio", "0.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-rollback-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-rollback"
					image    = "127.0.0.1:15000/tftest-service:v3"
					mode {
						replicated {
							replicas = 4
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "5s"
						failure_action    = "rollback"
						monitor           = "10s"
						max_failure_ratio = "0.0"
						order             = "stop-first"
					}

					rollback_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "4s"
						max_failure_ratio = "0.0"
						order             = "stop-first"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
						timeout  = "3m"
					}

					destroy_grace_seconds = "10"
				}
				`,
				ExpectError: regexp.MustCompile(`.*rollback completed.*`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-rollback"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3819682835.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "rollback"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.monitor", "4s"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.max_failure_ratio", "0.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateNetworksConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					networks = ["${docker_network.test_network.name}"]
					endpoint_mode = "vip"

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.2768978924", "tftest-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_mode", "vip"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					networks = ["${docker_network.test_network2.name}"]
					endpoint_mode = "vip"

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.3016497949", "tftest-network2"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_mode", "vip"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					networks = [
						"${docker_network.test_network.name}",
						"${docker_network.test_network2.name}"
					]
					endpoint_mode = "vip"

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "networks.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_mode", "vip"),
				),
			},
		},
	})
}
func TestAccDockerService_updateMountsConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_volume" "foo" {
					name = "tftest-volume"
				}

				resource "docker_volume" "foo2" {
					name = "tftest-volume2"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-mounts"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					mounts = [
						{
							source = "${docker_volume.foo.name}"
							target = "/mount/test"
							type   = "volume"
							read_only = true
							volume_labels {
								env = "dev"
								terraform = "true"
							}
						}
					]

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
					
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-mounts"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_volume" "foo" {
					name = "tftest-volume"
				}

				resource "docker_volume" "foo2" {
					name = "tftest-volume2"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-mounts"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					mounts = [
						{
							source = "${docker_volume.foo.name}"
							target = "/mount/test"
							type   = "volume"
							read_only = true
							volume_labels {
								env = "dev"
								terraform = "true"
							}
						},
						{
							source = "${docker_volume.foo2.name}"
							target = "/mount/test2"
							type   = "volume"
							read_only = true
							volume_labels {
								env = "dev"
								terraform = "true"
							}
						}
					]

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-mounts"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mounts.#", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateHostsConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					hosts = [
						{
							host = "testhost"
							ip = "10.0.1.0"
						}
					]

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}
					
					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					hosts = [
						{
							host = "testhost2"
							ip = "10.0.2.2"
						}
					]

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					hosts = [
						{
							host = "testhost"
							ip = "10.0.1.0"
						},
						{
							host = "testhost2"
							ip = "10.0.2.2"
						}
					]

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "hosts.#", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateLoggingConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					logging {
						driver_name = "json-file"
					
						options {
							max-size = "10m"
							max-file = "3"
						}
					}

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.driver_name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-size", "10m"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-file", "3"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					logging {
						driver_name = "json-file"
					
						options {
							max-size = "15m"
							max-file = "5"
						}
					}

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.driver_name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-size", "15m"),
					resource.TestCheckResourceAttr("docker_service.foo", "logging.0.options.max-file", "5"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					image    = "stovogel/friendlyhello:part2"
					mode {
						replicated {
							replicas = 2
						}
					}

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "stovogel/friendlyhello:part2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateHealthcheckConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-healthcheck-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-healthcheck"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-healthcheck"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-healthcheck-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-healthcheck"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-healthcheck"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateIncreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-increase-replicas-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-increase-replicas"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-increase-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					// resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-increase-replicas-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-increase-replicas"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 3
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-increase-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.2901027540.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-decrease-replicas-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-decrease-replicas"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 5
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-decrease-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.4205874514.replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-decrease-replicas-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-decrease-replicas"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-decrease-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					// resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

func TestAccDockerService_updateImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-image-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-image"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-image-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-image"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigAndSecretConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_secret" "service_secret" {
					name 			 = "tftest-tftest-mysecret-${replace(timestamp(),":", ".")}"
					data 			 = "ewogICJrZXkiOiAiUVdFUlRZIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-secret"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-secret"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name       = "tftest-myconfig-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_secret" "service_secret" {
					name       = "tftest-tftest-mysecret-${replace(timestamp(),":", ".")}"
					data       = "ewogICJrZXkiOiAiUVdFUlRZIgp9" # UPDATED to YXCVB

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-secret"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-secret"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updatePortConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-port-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-port"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-port"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name = "tftest-up-port-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-port"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 4
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-port"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3819682835.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigReplicasImageAndHealthConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihc"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihc"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihc"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 4
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihc"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3819682835.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigAndDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-dec-repl"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 5
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-dec-repl"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.4205874514.replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-dec-repl"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-dec-repl"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					// resource.TestCheckResourceAttr("docker_service.foo", "replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.1587501533.external", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigReplicasImageAndHealthIncreaseAndDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v1"
					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3052477333.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 6
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.3516784273.replicas", "6"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					image    = "127.0.0.1:15000/tftest-service:v2"
					mode {
						replicated {
							replicas = 3
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
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

					converge_config {
						interval = "500ms"
						monitor  = "10s"
					}

					destroy_grace_seconds = "10"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.2901027540.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.3668852780.external", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.internal", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "ports.2374790270.external", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

func TestAccDockerService_privateConverge(t *testing.T) {
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
						name     = "tftest-service-bar"
						image    = "%s"
						mode {
							replicated {
								replicas = 2
							}
						}
					}
				`, registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.bar", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.bar", "name", "tftest-service-bar"),
					resource.TestCheckResourceAttr("docker_service.bar", "image", image),
					// resource.TestCheckResourceAttr("docker_service.bar", "replicas", "2"),
				),
			},
		},
	})
}

// Helpers
func isServiceRemoved(serviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		filter := make(map[string][]string)
		filter["name"] = []string{serviceName}
		services, err := client.ListServices(dc.ListServicesOptions{
			Filters: filter,
		})
		if err != nil {
			return fmt.Errorf("Error listing service for name %s: %v", serviceName, err)
		}
		length := len(services)
		log.Printf("### isServiceRemoved length: %v", length)
		if length != 0 {
			return fmt.Errorf("Service should be removed but is running: %s", serviceName)
		}

		return nil
	}
}
