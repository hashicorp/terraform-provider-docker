package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*ProviderConfig).DockerClient
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
	client := meta.(*ProviderConfig).DockerClient

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return err
	}

	createOpts := dc.CreateServiceOptions{
		ServiceSpec: serviceSpec,
	}

	if v, ok := d.GetOk("auth"); ok {
		createOpts.Auth = authToServiceAuth(v.(map[string]interface{}))
	} else {
		createOpts.Auth = fromRegistryAuth(d.Get("image").(string), meta.(*ProviderConfig).AuthConfigs.Configs)
	}

	if v, ok := d.GetOk("update_config"); ok {
		createOpts.UpdateConfig, _ = createUpdateOrRollbackConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("rollback_config"); ok {
		createOpts.RollbackConfig, _ = createUpdateOrRollbackConfig(v.([]interface{}))
	}

	service, err := client.CreateService(createOpts)
	if err != nil {
		return err
	}

	configIDs := extractSetProperty(d, "configs", "config_id")
	secretIDs := extractSetProperty(d, "secrets", "secret_id")
	n := d.Get("replicas")
	if err := areAtLeastNContainersUp(d.Get("name").(string), d.Get("image").(string), service.ID, configIDs, secretIDs, n.(int), client); err != nil {
		return err
	}

	d.SetId(service.ID)
	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

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
	client := meta.(*ProviderConfig).DockerClient

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
	} else {
		updateOpts.Auth = fromRegistryAuth(d.Get("image").(string), meta.(*ProviderConfig).AuthConfigs.Configs)
	}

	err = client.UpdateService(d.Id(), updateOpts)
	if err != nil {
		return err
	}

	configIDs := extractSetProperty(d, "configs", "config_id")
	secretIDs := extractSetProperty(d, "secrets", "secret_id")
	n := d.Get("replicas")
	if err := areAtLeastNContainersUp(d.Get("name").(string), d.Get("image").(string), d.Id(), configIDs, secretIDs, n.(int), client); err != nil {
		return err
	}

	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	// == 1: get containerIDs of the running service
	serviceContainerIds := make([]string, 0)
	filter := make(map[string][]string)
	filter["service"] = []string{d.Get("name").(string)}
	tasks, err := client.ListTasks(dc.ListTasksOptions{
		Filters: filter,
	})
	if err != nil {
		return err
	}
	for i := 0; i < len(tasks); i++ {
		task, _ := client.InspectTask(tasks[i].ID)
		log.Printf("[INFO] Inspecting container '%s' and state '%s' for shutdown", task.Status.ContainerStatus.ContainerID, task.Status.State)
		if strings.TrimSpace(task.Status.ContainerStatus.ContainerID) != "" && task.Status.State != swarm.TaskStateShutdown {
			serviceContainerIds = append(serviceContainerIds, task.Status.ContainerStatus.ContainerID)
		}
	}

	// == 2: delete the service
	if err := deleteService(d.Id(), client); err != nil {
		return err
	}

	// == 3: wait until all containers of the service are down to be able to unmount the associated volumes
	for _, containerID := range serviceContainerIds {
		log.Printf("[INFO] Waiting for container: '%s' to exit max 1 minute", containerID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel() // releases resources if operation completes before timeout elapses
		if exitCode, errOnExit := client.WaitContainerWithContext(containerID, ctx); errOnExit == nil {
			log.Printf("[INFO] Container: '%s' exited with code '%d'", containerID, exitCode)
			select {
			case <-ctx.Done():
				// ctx is cancelled, kill the container
				log.Printf("[INFO] Container: '%s' shutdown was canceled. Killing it now", containerID)
				killOps := dc.KillContainerOptions{
					Signal:  dc.SIGKILL,
					Context: ctx,
				}
				if errOnKill := client.KillContainer(killOps); errOnKill != nil {
					log.Printf("[INFO] Container: '%s' killing errord with '%s'", containerID, errOnKill)
				}
			default:
				// ctx is not canceled, continue immediately
				log.Printf("[INFO] Container: '%s' shutdown was NOT canceled. Continueing", containerID)
			}
			// Remove the container if it did not exit properly to be able to unmount the volumes
			if exitCode != 0 {
				log.Printf("[INFO] Container: '%s' exited with non-null code '%d' -> removing it", containerID, exitCode)
				removeOps := dc.RemoveContainerOptions{
					ID:      containerID,
					Force:   true,
					Context: ctx, // 1 min timeout as well here
				}
				if errOnRemove := client.RemoveContainer(removeOps); errOnRemove != nil {
					// if the removal is already in progress, this error can be ignored
					log.Printf("[INFO] Error '%s' on removal of Container: '%s'", errOnRemove, containerID)
				}
			}
		}
	}

	d.SetId("")
	return nil
}

////////////
// Helpers
////////////
func deleteService(serviceID string, client *dc.Client) error {
	removeOpts := dc.RemoveServiceOptions{
		ID: serviceID,
	}

	if err := client.RemoveService(removeOpts); err != nil {
		return fmt.Errorf("Error deleting service %s: %s", serviceID, err)
	}

	return nil
}

func fetchDockerService(ID string, name string, client *dc.Client) (*swarm.Service, error) {
	apiServices, err := client.ListServices(dc.ListServicesOptions{})

	if err != nil {
		return nil, fmt.Errorf("Error fetching service information from Docker: %s", err)
	}

	for _, apiService := range apiServices {
		if apiService.ID == ID || apiService.Spec.Name == name {
			return &apiService, nil
		}
	}

	return nil, nil
}

func createServiceSpec(d *schema.ResourceData) (swarm.ServiceSpec, error) {

	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		TaskTemplate: swarm.TaskSpec{},
	}

	if v, ok := d.GetOk("replicas"); ok {
		replicas := uint64(v.(int))
		serviceSpec.Mode = swarm.ServiceMode{}
		serviceSpec.Mode.Replicated = &swarm.ReplicatedService{}
		serviceSpec.Mode.Replicated.Replicas = &replicas
	}

	// == start Container Spec
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

	if v, ok := d.GetOk("host"); ok {
		containerSpec.Hosts = extraHostsSetToDockerExtraHosts(v.(*schema.Set))
	}

	endpointSpec := swarm.EndpointSpec{}

	if v, ok := d.GetOk("network_mode"); ok {
		endpointSpec.Mode = swarm.ResolutionMode(v.(string))
	}

	portBindings := []swarm.PortConfig{}

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

	if v, ok := d.GetOk("mounts"); ok {
		mounts := []mount.Mount{}

		for _, rawMount := range v.(*schema.Set).List() {
			rawMount := rawMount.(map[string]interface{})
			mountType := mount.Type(rawMount["type"].(string))
			mountInstance := mount.Mount{
				Type:     mountType,
				Source:   rawMount["source"].(string),
				Target:   rawMount["target"].(string),
				ReadOnly: rawMount["read_only"].(bool),
			}

			if w, ok := d.GetOk("volume_labels"); ok {
				mountInstance.VolumeOptions.Labels = mapTypeMapValsToString(w.(map[string]interface{}))
			}

			if mountType == mount.TypeBind {
				if w, ok := d.GetOk("bind_propagation"); ok {
					mountInstance.BindOptions = &mount.BindOptions{
						Propagation: mount.Propagation(w.(string)),
					}
				}
			} else if mountType == mount.TypeVolume {
				mountInstance.VolumeOptions = &mount.VolumeOptions{}

				if w, ok := d.GetOk("volume_no_copy"); ok {
					mountInstance.VolumeOptions.NoCopy = w.(bool)
				}

				mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
				if w, ok := d.GetOk("volume_driver_name"); ok {
					mountInstance.VolumeOptions.DriverConfig.Name = w.(string)
				}

				if w, ok := d.GetOk("volume_driver_options"); ok {
					mountInstance.VolumeOptions.DriverConfig.Options = w.(map[string]string)
				}
			} else if mountType == mount.TypeTmpfs {
				mountInstance.TmpfsOptions = &mount.TmpfsOptions{}

				if w, ok := d.GetOk("tmpfs_size_bytes"); ok {
					mountInstance.TmpfsOptions.SizeBytes = w.(int64)
				}

				if w, ok := d.GetOk("tmpfs_mode"); ok {
					mountInstance.TmpfsOptions.Mode = os.FileMode(w.(int))
				}
			}

			mounts = append(mounts, mountInstance)
		}

		containerSpec.Mounts = mounts
	}

	serviceSpec.TaskTemplate.ContainerSpec = &containerSpec

	if v, ok := d.GetOk("configs"); ok {
		configs := []*swarm.ConfigReference{}

		for _, rawConfig := range v.(*schema.Set).List() {
			rawConfig := rawConfig.(map[string]interface{})
			config := swarm.ConfigReference{
				ConfigID:   rawConfig["config_id"].(string),
				ConfigName: rawConfig["config_name"].(string),
				File: &swarm.ConfigReferenceFileTarget{
					Name: rawConfig["file_name"].(string),
					GID:  "0",
					UID:  "0",
					Mode: os.FileMode(0444),
				},
			}
			configs = append(configs, &config)
		}
		serviceSpec.TaskTemplate.ContainerSpec.Configs = configs
	}

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
	// == end Container Spec

	// == start Endpoint Spec
	if v, ok := d.GetOk("ports"); ok {
		portBindings = portSetToServicePorts(v.(*schema.Set))
	}
	if len(portBindings) != 0 {
		endpointSpec.Ports = portBindings
	}

	serviceSpec.EndpointSpec = &endpointSpec
	// == end Endpoint Spec

	// == start TaskTemplate Spec
	placement := swarm.Placement{}
	if v, ok := d.GetOk("constraints"); ok {
		placement.Constraints = stringSetToStringSlice(v.(*schema.Set))
	}

	serviceSpec.TaskTemplate.Placement = &placement

	if v, ok := d.GetOk("logging"); ok {
		serviceSpec.TaskTemplate.LogDriver = &swarm.Driver{}
		for _, rawLogging := range v.([]interface{}) {
			rawLogging := rawLogging.(map[string]interface{})
			serviceSpec.TaskTemplate.LogDriver.Name = rawLogging["driver_name"].(string)

			if rawOptions, ok := rawLogging["options"]; ok {
				serviceSpec.TaskTemplate.LogDriver.Options = mapTypeMapValsToString(rawOptions.(map[string]interface{}))
			}
		}
	}

	if v, ok := d.GetOk("healthcheck"); ok {
		containerSpec.Healthcheck = &container.HealthConfig{}
		if len(v.([]interface{})) > 0 {
			for _, rawHealthCheck := range v.([]interface{}) {
				rawHealthCheck := rawHealthCheck.(map[string]interface{})
				if testCommand, ok := rawHealthCheck["test"]; ok {
					containerSpec.Healthcheck.Test = stringListToStringSlice(testCommand.([]interface{}))
				}
				if rawInterval, ok := rawHealthCheck["interval"]; ok {
					containerSpec.Healthcheck.Interval, _ = time.ParseDuration(rawInterval.(string))
				}
				if rawTimeout, ok := rawHealthCheck["timeout"]; ok {
					containerSpec.Healthcheck.Timeout, _ = time.ParseDuration(rawTimeout.(string))
				}
				if rawStartPeriod, ok := rawHealthCheck["start_period"]; ok {
					containerSpec.Healthcheck.StartPeriod, _ = time.ParseDuration(rawStartPeriod.(string))
				}
				if rawRetries, ok := rawHealthCheck["retries"]; ok {
					containerSpec.Healthcheck.Retries, _ = rawRetries.(int)
				}
			}
		}
	}
	// == end TaskTemplate Spec

	return serviceSpec, nil
}

func createUpdateOrRollbackConfig(config []interface{}) (*swarm.UpdateConfig, error) {
	updateConfig := swarm.UpdateConfig{}
	if len(config) > 0 {
		sc := config[0].(map[string]interface{})
		if v, ok := sc["parallelism"]; ok {
			updateConfig.Parallelism = uint64(v.(int))
		}
		if v, ok := sc["delay"]; ok {
			updateConfig.Delay, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["failure_action"]; ok {
			updateConfig.FailureAction = v.(string)
		}
		if v, ok := sc["monitor"]; ok {
			updateConfig.Monitor, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["max_failure_ratio"]; ok {
			updateConfig.MaxFailureRatio = float32(v.(float64))
		}
		if v, ok := sc["order"]; ok {
			updateConfig.Order = v.(string)
		}
	}

	return &updateConfig, nil
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
	if auth["username"] != nil && len(auth["username"].(string)) > 0 && auth["password"] != nil && len(auth["password"].(string)) > 0 {
		return dc.AuthConfiguration{
			Username:      auth["username"].(string),
			Password:      auth["password"].(string),
			ServerAddress: auth["server_address"].(string),
		}
	}

	return dc.AuthConfiguration{}
}

func fromRegistryAuth(image string, configs map[string]dc.AuthConfiguration) dc.AuthConfiguration {
	// Remove normalized prefixes to simlify substring
	image = strings.Replace(strings.Replace(image, "http://", "", 1), "https://", "", 1)
	// Get the registry with optional port
	lastBin := strings.Index(image, "/")
	// No auth given and image name has no slash like 'alpine:3.1'
	if lastBin != -1 {
		serverAddress := image[0:lastBin]
		if fromRegistryAuth, ok := configs[normalizeRegistryAddress(serverAddress)]; ok {
			return fromRegistryAuth
		}
	}

	return dc.AuthConfiguration{}
}

func getAmountOfTasksWithImageConfigAndSecret(tasks []swarm.Task, image string, configIDs []string, secretIDs []string) int {
	amount := 0
	for _, task := range tasks {
		// 1: check for images updates
		if task.Spec.ContainerSpec.Image == image &&
			isConfigIDPresent(task.Spec.ContainerSpec.Configs, configIDs) &&
			isSecretIDPresent(task.Spec.ContainerSpec.Secrets, secretIDs) {
			// 2: check for config and secret updates/additions
			amount++
		}
	}
	return amount
}

func isConfigIDPresent(configs []*swarm.ConfigReference, configIDs []string) bool {
	if len(configs) == 0 || len(configIDs) == 0 {
		log.Printf("[INFO] NO configID presence to perform") // TODO
		return true
	}

	for _, config := range configs {
		for _, configID := range configIDs {
			if (*config).ConfigID == configID {
				log.Printf("[INFO] configID '%s is present", configID)
				return true
			}
		}
	}
	return false
}

func isSecretIDPresent(secrets []*swarm.SecretReference, secretIDs []string) bool {
	if len(secrets) == 0 || len(secretIDs) == 0 {
		log.Printf("[INFO] NO secretID presence to perform")
		return true
	}

	for _, secret := range secrets {
		for _, secretID := range secretIDs {
			if (*secret).SecretID == secretID {
				log.Printf("[INFO] secretID '%s is present", secretID)
				return true
			}
		}
	}
	return false
}

func extractSetProperty(d *schema.ResourceData, setKey string, key string) []string {
	properties := make([]string, 0)
	if givenSet, ok := d.GetOk(setKey); ok {
		for _, rawSet := range givenSet.(*schema.Set).List() {
			rawSet := rawSet.(map[string]interface{})
			if value, ok := rawSet[key]; ok {
				log.Printf("[INFO] Found propery '%s' in set '%s'", key, setKey)
				properties = append(properties, value.(string))
			}
		}
	}
	return properties
}

func areAtLeastNContainersUp(serviceName string, image string, serviceID string, configIDs []string, secretIDs []string, n int, client *dc.Client) error {
	// config
	loops := 240
	sleepTime := 1000 * time.Millisecond
	maxErrorCount := 3

	// == 1: get at least n tasks for the given service name
	taskIDs := make([]string, 0)
	filter := make(map[string][]string)
	filter["service"] = []string{serviceName}
	filter["desired-state"] = []string{"running"}

	for i := 1; i <= loops; i++ {
		tasks, err := client.ListTasks(dc.ListTasksOptions{
			Filters: filter,
		})
		if err != nil {
			return err
		}
		amountOfTasksWithImageConfigAndSecret := getAmountOfTasksWithImageConfigAndSecret(tasks, image, configIDs, secretIDs)
		if amountOfTasksWithImageConfigAndSecret == n {
			for _, task := range tasks {
				taskIDs = append(taskIDs, task.ID)
			}
			log.Printf("[INFO] Got at least %d running task(s) for service '%s' and image '%s' after %d seconds", n, serviceName, image, i)
			break
		}
		log.Printf("[INFO] Service '%s' task loop: %02d/%d for amount of registered tasks %02d/%d", serviceName, i, loops, amountOfTasksWithImageConfigAndSecret, n)
		time.Sleep(sleepTime)
	}

	// no running task found -> deleting service
	if len(taskIDs) == 0 {
		log.Printf("[INFO] Found no running task for service '%s' after %d seconds", serviceName, loops)
		deleteService(serviceID, client)
		return fmt.Errorf("[INFO] Deleted service '%s' due to no task registration", serviceName)
	}

	// == 2: inspect that n tasks are up
	for _, taskID := range taskIDs {
		errorCount := 0
		for i := 1; i <= loops; i++ {
			task, err := client.InspectTask(taskID)
			// handle special case of decreasing amount of replicas
			// then the taskID may not exist any more at this point
			if err != nil {
				if strings.Contains(err.Error(), "No such task") {
					log.Printf("[INFO] No such task for taskID '%s'. Going to next one", taskID)
					break // go to next task
				}
				return err // all other errors should be reported
			}
			log.Printf("[INFO] Inspecting task with ID '%s' in loop %02d/%d to be in state: [%s]->[%s]", taskID, i, loops, task.Status.State, task.DesiredState)
			if task.DesiredState == task.Status.State {
				log.Printf("[INFO] Task '%s' containerID '%s' is in desired state '%s'", taskID, task.Status.ContainerStatus.ContainerID, task.DesiredState)
				break // success here, next task
			}

			if task.Status.State == swarm.TaskStateFailed ||
				task.Status.State == swarm.TaskStateRejected ||
				task.Status.State == swarm.TaskStateShutdown {
				errorCount++
				log.Printf("[INFO] Task '%s' containerID '%s' is in error state '%s' for %d/%d", taskID, task.Status.ContainerStatus.ContainerID, task.Status.State, errorCount, maxErrorCount)
				if errorCount >= maxErrorCount {
					log.Printf("[INFO] Task '%s' for service '%s' in state '%s' after %d errors. Deleting service!", taskID, serviceName, task.Status.State, maxErrorCount)
					deleteService(serviceID, client)
					return fmt.Errorf("[INFO] Deleted service '%s' due to not all tasks were up", serviceName)
				}
			}
			time.Sleep(sleepTime) // sleep between inspecting tasks
		}
	}

	log.Printf("[INFO] For service '%s' and image '%s' desired '%d' replica(s) with configIDs: '%v' and secretIDs: %v are up!", serviceName, image, n, configIDs, secretIDs)
	return nil
}
