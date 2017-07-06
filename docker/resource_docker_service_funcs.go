package docker

import (
	"fmt"
	"time"

	"os"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerServiceCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*dc.Client)

	createOpts := dc.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: d.Get("name").(string),
			},
			TaskTemplate: swarm.TaskSpec{},
		},
	}

	placement := swarm.Placement{}

	containerSpec := swarm.ContainerSpec{
		Image: d.Get("image").(string),
	}

	if v, ok := d.GetOk("hostname"); ok {
		containerSpec.Hostname = v.(string)
	}

	if v, ok := d.GetOk("command"); ok {
		containerSpec.Command = stringListToStringSlice(v.([]interface{}))
		for _, v := range containerSpec.Command {
			if v == "" {
				return fmt.Errorf("values for command may not be empty")
			}
		}
	}

	if v, ok := d.GetOk("env"); ok {
		containerSpec.Env = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("hosts"); ok {
		containerSpec.Hosts = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("constraints"); ok {
		placement.Constraints = stringSetToStringSlice(v.(*schema.Set))
	}

	endpointSpec := swarm.EndpointSpec{}

	if v, ok := d.GetOk("network_mode"); ok {
		endpointSpec.Mode = swarm.ResolutionMode(v.(string))
	}

	portBindings := []swarm.PortConfig{}

	if v, ok := d.GetOk("ports"); ok {
		portBindings = portSetToServicePorts(v.(*schema.Set))
	}
	if len(portBindings) != 0 {
		endpointSpec.Ports = portBindings
	}

	if v, ok := d.GetOk("networks"); ok {
		networks := []swarm.NetworkAttachmentConfig{}

		for _, rawNetwork := range v.(*schema.Set).List() {
			network := swarm.NetworkAttachmentConfig{
				Target: rawNetwork.(string),
			}
			networks = append(networks, network)
		}
		createOpts.ServiceSpec.TaskTemplate.Networks = networks
		createOpts.ServiceSpec.Networks = networks
	}

	createOpts.ServiceSpec.TaskTemplate.ContainerSpec = containerSpec

	if v, ok := d.GetOk("secrets"); ok {
		secrets := []*swarm.SecretReference{}

		for _, rawSecret := range v.(*schema.Set).List() {
			rawSecret := rawSecret.(map[string]interface{})
			secret := swarm.SecretReference{
				SecretID:   rawSecret["secret_id"].(string),
				SecretName: rawSecret["secret_name"].(string),
				File: &swarm.SecretReferenceFileTarget{
					Name: rawSecret["file_name"].(string),
					GID:  "0",
					UID:  "0",
					Mode: os.FileMode(0444),
				},
			}
			secrets = append(secrets, &secret)
		}
		createOpts.ServiceSpec.TaskTemplate.ContainerSpec.Secrets = secrets
	}

	createOpts.ServiceSpec.EndpointSpec = &endpointSpec

	createOpts.ServiceSpec.TaskTemplate.Placement = &placement

	if v, ok := d.GetOk("auth"); ok {
		createOpts.Auth = authToServiceAuth(v.(map[string]interface{}))
	}

	service, err := client.CreateService(createOpts)

	if err != nil {
		return err
	}

	creationTime = time.Now()
	d.SetId(service.ID)

	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	apiService, err := fetchDockerService(d.Id(), client)
	if err != nil {
		return err
	}
	if apiService == nil {
		// This service doesn't exist anymore
		d.SetId("")
		return nil
	}

	var service *swarm.Service
	service, err = client.InspectService(apiService.ID)
	if err != nil {
		return fmt.Errorf("Error inspecting service %s: %s", apiService.ID, err)
	}

	d.Set("version", service.Version)

	return nil
}

func resourceDockerServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDockerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	removeOpts := dc.RemoveServiceOptions{
		ID: d.Id(),
	}

	if err := client.RemoveService(removeOpts); err != nil {
		return fmt.Errorf("Error deleting service %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func fetchDockerService(ID string, client *dc.Client) (*swarm.Service, error) {
	apiServices, err := client.ListServices(dc.ListServicesOptions{})

	if err != nil {
		return nil, fmt.Errorf("Error fetching service information from Docker: %s\n", err)
	}

	for _, apiService := range apiServices {
		if apiService.ID == ID {
			return &apiService, nil
		}
	}

	return nil, nil
}

func portSetToServicePorts(ports *schema.Set) []swarm.PortConfig {
	retPortConfigs := []swarm.PortConfig{}

	for _, portInt := range ports.List() {
		port := portInt.(map[string]interface{})
		internal := port["internal"].(int)
		protocol := port["protocol"].(string)
		external := port["external"].(int)

		portConfig := swarm.PortConfig{
			TargetPort:    uint32(internal),
			PublishedPort: uint32(external),
			Protocol:      swarm.PortConfigProtocol(protocol),
		}

		mode, modeOk := port["mode"].(string)
		if modeOk {
			portConfig.PublishMode = swarm.PortConfigPublishMode(mode)
		}

		retPortConfigs = append(retPortConfigs, portConfig)
	}

	return retPortConfigs
}

func authToServiceAuth(auth map[string]interface{}) dc.AuthConfiguration {
	return dc.AuthConfiguration{
		Username:      auth["username"].(string),
		Password:      auth["password"].(string),
		ServerAddress: auth["server_address"].(string),
	}
}
