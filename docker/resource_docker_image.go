package docker

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerImageCreate,
		Read:   resourceDockerImageRead,
		Update: resourceDockerImageUpdate,
		Delete: resourceDockerImageDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"latest": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"keep_locally": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"pull_trigger": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"pull_triggers"},
				Deprecated:    "Use field pull_triggers instead",
			},

			"pull_triggers": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"output": {
				Type:     schema.TypeString,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"build": {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"pull_triggers", "pull_trigger"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:        schema.TypeString,
							Description: "Context path",
							Required:    true,
						},
						"dockerfile": {
							Type:        schema.TypeString,
							Description: "Name of the Dockerfile (Default is 'PATH/Dockerfile')",
							Optional:    true,
							Default:     "Dockerfile",
						},
						"tag": {
							Type:        schema.TypeString,
							Description: "Name and optionally a tag in the 'name:tag' format",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}
