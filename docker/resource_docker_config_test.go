package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDockerConfig_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		// swarm will be initialized in 'testAccPreCheck' if necessary
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "foo" {
					name = "foo-${replace(timestamp(),":", ".")}"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="

					lifecycle {
						ignore_changes = ["name"]
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					// resource.TestCheckResourceAttr("docker_config.foo", "name", "foo"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
		},
	})
}
