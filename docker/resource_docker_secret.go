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
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},

			// Workaround until https://github.com/moby/moby/issues/35803 is fixed
			"updateable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
		},
	}
}
