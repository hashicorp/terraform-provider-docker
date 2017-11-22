package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDockerSecret_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_secret" "foo" {
					name = "foo"
					data = "ewodwerwefdvweew4534gICJzZXJ2ZXZZ67IiOiB7CiA="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_secret.foo", "name", "foo"),
					resource.TestCheckResourceAttr("docker_secret.foo", "data", "ewodwerwefdvweew4534gICJzZXJ2ZXZZ67IiOiB7CiA="),
				),
			},
		},
	})
}
