package docker

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerNetworkCreate,
		Read:   resourceDockerNetworkRead,
		Delete: resourceDockerNetworkDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"check_duplicate": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "bridge",
			},

			"options": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"internal": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"attachable": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ingress": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"ipam_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "default",
			},

			"ipam_config": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"ip_range": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"gateway": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"aux_address": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
