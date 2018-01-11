package docker

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerConfig_basicNotUpdateable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		// swarm will be initialized in 'testAccPreCheck' if necessary
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckDockerConfigDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "foo" {
					name = "foo-config"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "name", "foo-config"),
					resource.TestCheckResourceAttr("docker_config.foo", "updateable", "false"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
		},
	})
}
func TestAccDockerConfig_basicUpdateable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		// swarm will be initialized in 'testAccPreCheck' if necessary
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckDockerConfigShouldStillExist,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "foo" {
					name 			 = "foo-${replace(timestamp(),":", ".")}"
					data 			 = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
					updateable = true

					lifecycle {
						ignore_changes = ["name"]
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					// resource.TestCheckResourceAttr("docker_config.foo", "name", "foo"),
					resource.TestCheckResourceAttr("docker_config.foo", "updateable", "true"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
		},
	})
}

/////////////
// Helpers
/////////////
func testCheckDockerConfigDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "configs" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		config, err := client.InspectConfig(id)

		if err == nil || config != nil {
			return fmt.Errorf("Config with id '%s' still exists", id)
		}
		return nil
	}
	return nil
}

func testCheckDockerConfigShouldStillExist(s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "configs" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		config, err := client.InspectConfig(id)

		if err != nil || config == nil {
			return fmt.Errorf("Config with id '%s' is destroyed but it should exist", id)
		}
		return nil
	}
	return nil
}
