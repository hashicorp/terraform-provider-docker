package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerService_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "friendlyhello" {
					name     = "service-foo"
					image    = "alpine:3.1"
					replicas = 10
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
