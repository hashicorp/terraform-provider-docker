package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerSecretCreate,
		Read:   resourceDockerSecretRead,
		Update: resourceDockerSecretUpdate,
		Delete: resourceDockerSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"data": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
				ForceNew:  true,
			},
		},
	}
}
