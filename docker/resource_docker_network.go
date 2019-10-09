package docker

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
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
				Computed: true,
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
			},

			"ipam_config": {
				Type:     schema.TypeSet,
				Optional: true,
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
				Set: resourceDockerIpamConfigHash,
			},

			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDockerIpamConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["subnet"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ip_range"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["gateway"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["aux_address"]; ok {
		auxAddress := v.(map[string]interface{})

		keys := make([]string, len(auxAddress))
		i := 0
		for k := range auxAddress {
			keys[i] = k
			i++
		}
		sort.Strings(keys)

		for _, k := range keys {
			buf.WriteString(fmt.Sprintf("%v-%v-", k, auxAddress[k].(string)))
		}
	}

	return hashcode.String(buf.String())
}
