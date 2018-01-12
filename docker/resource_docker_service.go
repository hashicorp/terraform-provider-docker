package docker

import (
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
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
						},
						"password": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Sensitive:   true,
						},
					},
				},
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
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validateIntegerGeqThan(1),
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
			"host": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"network_mode": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
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
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateStringMatchesPattern(`^(bind|volume|tmpf)$`),
						},
						"consistency": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern(`^(default|consistent|cached|delegated)$`),
						},
						"read_only": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"bind_propagation": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
						},
						"volume_no_copy": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"volume_labels": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
							Optional: true,
						},
						"file_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
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
							Type:         schema.TypeString,
							Default:      "vip",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
						},
						"protocol": &schema.Schema{
							Type:         schema.TypeString,
							Default:      "tcp",
							Optional:     true,
							ValidateFunc: validateStringMatchesPattern(`^(tcp|udp)$`),
						},
					},
				},
			},

			"update_config": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("^(pause|continue|rollback)$"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeFloat,
							Optional:     true,
							Default:      0.0,
							ValidateFunc: validateFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("^(stop-first|start-first)$"),
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
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between task rollbacks (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on rollback failure: pause | continue | rollback",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("(pause|continue|rollback)"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeFloat,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							Default:      0.0,
							ValidateFunc: validateFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Rollback order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
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

			"logging": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"driver_name": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateStringMatchesPattern("(none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs)"),
						},
						"options": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"healthcheck": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"interval": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "10s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "3s",
							ValidateFunc: validateDurationGeq0(),
						},
						"start_period": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "2s",
							ValidateFunc: validateDurationGeq0(),
						},
						"retries": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},
		},
	}
}
