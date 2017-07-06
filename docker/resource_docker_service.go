package docker

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerService() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerServiceCreate,
		Read:   resourceDockerServiceRead,
		Update: resourceDockerServiceUpdate,
		Delete: resourceDockerServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ports": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"external": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},

						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "ingress",
							Optional: true,
							ForceNew: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerPortsHash,
			},

			"env": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"hosts": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"constraints": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"network_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"networks": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"secrets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"secret_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"secret_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"file_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerSecretsHash,
			},

			"auth": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"password": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},

						"server_address": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "ingress",
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerAuthHash,
			},
		},
	}
}

func resourceDockerAuthHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%v-", m["username"].(int)))

	if v, ok := m["password"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(int)))
	}

	if v, ok := m["server_address"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}
