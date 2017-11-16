package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerConfig_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "foo" {
					name = "foo"
					data = "ewodwerwefdvweew4534gICJzZXJ2ZXZZ67IiOiB7CiA="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						return nil
						// return fmt.Errorf("err")
					}),
			},
		},
	})
}
