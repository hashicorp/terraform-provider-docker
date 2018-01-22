package docker

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerConfigCreate,
		Read:   resourceDockerConfigRead,
		Update: resourceDockerConfigUpdate,
		Delete: resourceDockerConfigDelete,

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
