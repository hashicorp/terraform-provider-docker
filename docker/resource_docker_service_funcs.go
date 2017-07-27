package docker

import (
	"fmt"
	"time"

	"os"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*dc.Client)
	if client == nil {
		return false, nil
	}

	apiService, err := fetchDockerService(d.Id(), d.Get("name").(string), client)
	if err != nil {
		return false, err
	}
	if apiService == nil {
		return false, nil
	}

	return true, nil
}

func resourceDockerServiceCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*dc.Client)

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return err
	}

	createOpts := dc.CreateServiceOptions{
		ServiceSpec: serviceSpec,
	}

	if v, ok := d.GetOk("auth"); ok {
		createOpts.Auth = authToServiceAuth(v.(map[string]interface{}))
	}

	service, err := client.CreateService(createOpts)
	if err != nil {
		return err
	}

	filter := make(map[string][]string)
	filter["service"] = []string{d.Get("name").(string)}

	taskID := ""
	errorCount := 0
	loops := 900
	sleepTime := 1000 * time.Millisecond
	for i := loops; i > 0; i-- {
		if taskID == "" {
			tasks, err := client.ListTasks(dc.ListTasksOptions{
				Filters: filter,
			})
			if err != nil {
				return err
			}
			if len(tasks) == 1 {
				taskID = tasks[0].ID
			} else {
				time.Sleep(sleepTime)
				continue
			}
		}

		task, err := client.InspectTask(taskID)
		if err != nil {
			return err
		}

		if task.DesiredState == task.Status.State {
			break
		}

		if task.Status.State == swarm.TaskStateFailed {
			errorCount++
			taskID = ""
			if errorCount >= 3 {
				return fmt.Errorf("Failed to start container: %s", task.Status.Err)
			}
		}

		time.Sleep(sleepTime)
	}

	d.SetId(service.ID)

	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	apiService, err := fetchDockerService(d.Id(), d.Get("name").(string), client)
	if err != nil {
		return err
	}
	if apiService == nil {
		d.SetId("")
		return nil
	}

	var service *swarm.Service
	service, err = client.InspectService(apiService.ID)
	if err != nil {
		return fmt.Errorf("Error inspecting service %s: %s", apiService.ID, err)
	}

	d.Set("version", service.Version.Index)
	d.Set("name", service.Spec.Name)

	d.SetId(service.ID)

	return nil
}

func resourceDockerServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	service, err := client.InspectService(d.Id())
	if err != nil {
		return err
	}

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return err
	}

	updateOpts := dc.UpdateServiceOptions{
		ServiceSpec: serviceSpec,
		Version:     service.Version.Index,
	}

	if v, ok := d.GetOk("auth"); ok {
		updateOpts.Auth = authToServiceAuth(v.(map[string]interface{}))
	}

	err = client.UpdateService(d.Id(), updateOpts)
	if err != nil {
		return err
	}

	loops := 30
	sleepTime := 500 * time.Millisecond
	for i := loops; i > 0; i-- {
		service, err := client.InspectService(d.Id())
		if err != nil {
			return err
		}

		if service.UpdateStatus.State == swarm.UpdateStateCompleted {
			break
		}

		if service.UpdateStatus.State == swarm.UpdateStateRollbackCompleted {
			return fmt.Errorf("Failed update service rolled back: %s", service.UpdateStatus.Message)
		}

		time.Sleep(sleepTime)
	}

	return resourceDockerServiceRead(d, meta)
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

func fetchDockerService(ID string, name string, client *dc.Client) (*swarm.Service, error) {
	apiServices, err := client.ListServices(dc.ListServicesOptions{})

	if err != nil {
		return nil, fmt.Errorf("Error fetching service information from Docker: %s\n", err)
	}

	for _, apiService := range apiServices {
		if apiService.ID == ID || apiService.Spec.Name == name {
			return &apiService, nil
		}
	}

	return nil, nil
}

func createServiceSpec(d *schema.ResourceData) (swarm.ServiceSpec, error) {
	placement := swarm.Placement{}

	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		TaskTemplate: swarm.TaskSpec{},
	}

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
				return swarm.ServiceSpec{}, fmt.Errorf("values for command may not be empty")
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
		serviceSpec.TaskTemplate.Networks = networks
		serviceSpec.Networks = networks
	}

	serviceSpec.TaskTemplate.ContainerSpec = containerSpec

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
		serviceSpec.TaskTemplate.ContainerSpec.Secrets = secrets
	}

	serviceSpec.EndpointSpec = &endpointSpec

	serviceSpec.TaskTemplate.Placement = &placement

	return serviceSpec, nil
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
