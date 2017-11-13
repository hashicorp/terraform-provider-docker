package docker

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceDockerService create a docker service
// https://github.com/moby/moby/blob/master/api/types/swarm/service.go
func resourceDockerService() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerServiceCreate,
		Read:   resourceDockerServiceRead,
		Update: resourceDockerServiceUpdate,
		Delete: resourceDockerServiceDelete,
		Exists: resourceDockerServiceExists,

		Schema: map[string]*schema.Schema{
			"auth": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"username": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerAuthHash,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// == start Container Spec
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"replicas": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"env": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"hosts": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"network_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"networks": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"mounts": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"source": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"read_only": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"bind_propagation": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"volume_no_copy": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"volume_driver_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"volume_driver_options": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"tmpfs_size_bytes": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"tmpfs_mode": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: resourceDockerMountsHash,
			},

			"configs": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"config_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"config_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"file_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceDockerConfigsHash,
			},

			"secrets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"secret_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"secret_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"file_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceDockerSecretsHash,
			},
			// == end Container Spec

			"ports": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"external": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "ingress",
							Optional: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
						},
					},
				},
				Set: resourceDockerPortsHash,
			},

			"update_config": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be updated in one iteration simultaneously (0 to update all at once)",
							Optional:     true,
							ValidateFunc: validateIntegerGeqThan0(),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between updates (ns|us|ms|s|m|h)",
							Optional:     true,
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on update failure: pause | continue | rollback",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern("(pause|continue|rollback)"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeFloat,
							Description:  "Failure rate to tolerate during an update",
							Optional:     true,
							ValidateFunc: validateFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Update order either 'stop-first' or 'start-first'",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern("(stop-first|start-first)"),
						},
					},
				},
			},

			"rollback_config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be rollbacked in one iteration",
							Optional:     true,
							ValidateFunc: validateIntegerGeqThan0(),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between task rollbacks (ns|us|ms|s|m|h)",
							Optional:     true,
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on rollback failure: pause | continue | rollback",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern("(pause|continue|rollback)"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeFloat,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							ValidateFunc: validateFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Rollback order: either 'stop-first' or 'start-first'",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern("(stop-first|start-first)"),
						},
					},
				},
			},

			"constraints": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceDockerAuthHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%v-", m["server_address"].(string)))

	if v, ok := m["username"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}
	if v, ok := m["password"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceDockerMountsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["target"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["source"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["type"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}
