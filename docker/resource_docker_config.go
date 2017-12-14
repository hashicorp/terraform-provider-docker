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
			// "docker_config_id": &schema.Schema{
			// 	Type:     schema.TypeString,
			// 	Computed: true,
			// 	// ComputedWhen: ["data"] not yet implemented: https://github.com/hashicorp/terraform/pull/4846
			// 	// Hence no recomputation is triggered when 'data' is changed and depending resources are not triggered
			// },
			"data": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},
		},
	}
}
