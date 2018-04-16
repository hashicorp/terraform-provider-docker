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
				Type:        schema.TypeString,
				Description: "Name of the service",
				Required:    true,
				ForceNew:    true,
			},
			// == start Container Spec
			"image": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The image name to use for the containers of the service",
				Required:    true,
			},
			"mode": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Scheduling mode for the service",
				MaxItems:    1,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicated": &schema.Schema{
							Type:          schema.TypeSet,
							Description:   "The replicated service mode",
							Optional:      true,
							ConflictsWith: []string{"mode.0.global"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"replicas": &schema.Schema{
										Type:         schema.TypeInt,
										Description:  "The amount of replicas of the service",
										Optional:     true,
										ValidateFunc: validateIntegerGeqThan(1),
									},
								},
							},
						},
						"global": &schema.Schema{
							Type:          schema.TypeBool,
							Description:   "The global service mode",
							Optional:      true,
							ConflictsWith: []string{"mode.0.replicated", "converge_config"},
						},
					},
				},
			},
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The hostname to use for the container, as a valid RFC 1123 hostname",
				Optional:    true,
			},
			"destroy_grace_seconds": &schema.Schema{
				Type:         schema.TypeInt,
				Description:  "Amount of seconds to wait for the container to terminate before forcefully removing it",
				Optional:     true,
				ValidateFunc: validateIntegerGeqThan(0),
			},
			"command": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The command to be run in the image.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"env": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "A list of environment variables in the form VAR=value",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"hosts": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "A list of hostname/IP mappings to add to the container's hosts file.",
				Optional:    true,
				ForceNew:    true,
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
			"endpoint_mode": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "The mode of resolution to use for internal load balancing between tasks",
				Optional:     true,
				Default:      "vip",
				ValidateFunc: validateStringMatchesPattern(`^(vip|dnsrr)$`),
			},
			"networks": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Ids of the networks in which the  container will be put in.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"mounts": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Specification for mounts to be added to containers created as part of the service",
				Optional:    true,
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
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
				Type:        schema.TypeSet,
				Description: "References to zero or more configs that will be exposed to the service",
				Optional:    true,
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
				Type:        schema.TypeSet,
				Description: "References to zero or more secrets that will be exposed to the service",
				Optional:    true,
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
				Type:        schema.TypeSet,
				Description: "List of exposed ports that this service is accessible on from the outside. Ports can only be provided if vip resolution mode is used.",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": &schema.Schema{
							Type:        schema.TypeInt,
							Description: "The port inside the container",
							Required:    true,
						},
						"external": &schema.Schema{
							Type:        schema.TypeInt,
							Description: "The port on the swarm hosts",
							Optional:    true,
						},
						"publish_mode": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "ingress",
							ValidateFunc: validateStringMatchesPattern(`^(host|ingress)$`),
						},
						"protocol": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "tcp or udp",
							Optional:     true,
							Default:      "tcp",
							ValidateFunc: validateStringMatchesPattern(`^(tcp|udp)$`),
						},
					},
				},
			},

			"update_config": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Specification for the update strategy of the service",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"parallelism": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Maximum number of tasks to be updated in one iteration",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Delay between task updates (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "0s",
							ValidateFunc: validateDurationGeq0(),
						},
						"failure_action": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Action on update failure: pause | continue | rollback",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("^(pause|continue|rollback)$"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during an update",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
						},
						"order": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Update order: either 'stop-first' or 'start-first'",
							Optional:     true,
							Default:      "stop-first",
							ValidateFunc: validateStringMatchesPattern("^(stop-first|start-first)$"),
						},
					},
				},
			},

			"rollback_config": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Specification for the rollback strategy of the service",
				Optional:    true,
				MaxItems:    1,
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
							Description:  "Action on rollback failure: pause | continue",
							Optional:     true,
							Default:      "pause",
							ValidateFunc: validateStringMatchesPattern("(pause|continue)"),
						},
						"monitor": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)",
							Optional:     true,
							Default:      "5s",
							ValidateFunc: validateDurationGeq0(),
						},
						"max_failure_ratio": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Failure rate to tolerate during a rollback",
							Optional:     true,
							Default:      "0.0",
							ValidateFunc: validateStringIsFloatRatio(),
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

			"placement": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The placement preferences",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"constraints": &schema.Schema{
							Type:        schema.TypeSet,
							Description: "An array of constraints. e.g.: node.role==manager",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"prefs": &schema.Schema{
							Type:        schema.TypeSet,
							Description: "Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence, e.g.: spread=node.role.manager",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"platforms": &schema.Schema{
							Type:        schema.TypeSet,
							Description: "Platforms stores all the platforms that the service's image can run om",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"architecture": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"os": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},

			"logging": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Describes the logging for the tasks of this service",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"driver_name": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "The logging driver to use: one of none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs",
							Required:     true,
							ValidateFunc: validateStringMatchesPattern("(none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs)"),
						},
						"options": &schema.Schema{
							Type:        schema.TypeMap,
							Description: "The options for the logging driver",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"healthcheck": &schema.Schema{
				Type:        schema.TypeList,
				Description: "A test to perform to check that the container is healthy",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": &schema.Schema{
							Type:        schema.TypeList,
							Description: "The test to perform as list",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Time between running the check (ms|s|m|h)",
							Optional:     true,
							Default:      "10s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Maximum time to allow one check to run (ms|s|m|h)",
							Optional:     true,
							Default:      "3s",
							ValidateFunc: validateDurationGeq0(),
						},
						"start_period": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)",
							Optional:     true,
							Default:      "2s",
							ValidateFunc: validateDurationGeq0(),
						},
						"retries": &schema.Schema{
							Type:         schema.TypeInt,
							Description:  "Consecutive failures needed to report unhealthy",
							Optional:     true,
							Default:      1,
							ValidateFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},

			"dns_config": &schema.Schema{
				Type:        schema.TypeList,
				Description: "Specification for DNS related configurations in resolver configuration file (resolv.conf)",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"nameservers": &schema.Schema{
							Type:        schema.TypeList,
							Description: "The IP addresses of the name servers",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"search": &schema.Schema{
							Type:        schema.TypeList,
							Description: "A search list for host-name lookup",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"options": &schema.Schema{
							Type:        schema.TypeList,
							Description: "A list of internal resolver variables to be modified (e.g., debug, ndots:3, etc.)",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"converge_config": &schema.Schema{
				Type:          schema.TypeList,
				Description:   "A configuration to ensure that a service converges aka reaches the desired that of all task up and running",
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{"mode.0.global"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delay": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "The interval to check if the desired state is reached (ms|s). Default: 7s",
							Optional:     true,
							Default:      "7s",
							ValidateFunc: validateDurationGeq0(),
						},
						"timeout": &schema.Schema{
							Type:         schema.TypeString,
							Description:  "The timeout of the service to reach the desired state (s|m). Default: 3m",
							Optional:     true,
							Default:      "3m",
							ValidateFunc: validateDurationGeq0(),
						},
					},
				},
			},
		},
	}
}
