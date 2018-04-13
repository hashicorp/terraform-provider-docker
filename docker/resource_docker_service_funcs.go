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
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type convergeConfig struct {
	interval   time.Duration
	monitor    time.Duration
	timeout    time.Duration
	timeoutRaw string
}

/////////////////
// TF CRUD funcs
/////////////////
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
	d.SetId(service.ID)

	if v, ok := d.GetOk("converge_config"); ok {
		convergeConfig := createConvergeConfig(v.([]interface{}))
		log.Printf("[INFO] Waiting for Service '%s' to be created with timeout: %v", service.ID, convergeConfig.timeoutRaw)
		timeout, _ := time.ParseDuration(convergeConfig.timeoutRaw)
		stateConf := &resource.StateChangeConf{
			Pending:    serviceCreatePendingStates,
			Target:     []string{"running", "complete"},
			Refresh:    resourceDockerServiceCreateRefreshFunc(service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      7 * time.Second,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForState()
		if err != nil {
			// the service will be deleted in case it cannot be converged
			if deleteErr := deleteService(service.ID, d, client); deleteErr != nil {
				return deleteErr
			}
			if strings.Contains(err.Error(), "timeout while waiting for state") {
				return &DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout}
			}
			return err
		}
	}

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

	service, err := client.InspectService(apiService.ID)
	if err != nil {
		return fmt.Errorf("Error inspecting service %s: %s", apiService.ID, err)
	}

	d.Set("name", service.Spec.Name)
	d.Set("image", service.Spec.TaskTemplate.ContainerSpec.Image)
	// TODO nesting
	// err = d.Set("mode", flattenServiceMode(service.Spec.Mode))
	// if err != nil {
	// 	return err
	// }
	if len(service.Spec.TaskTemplate.ContainerSpec.Hostname) > 0 {
		err = d.Set("hostname", service.Spec.TaskTemplate.ContainerSpec.Hostname)
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Command) > 0 {
		err = d.Set("command", service.Spec.TaskTemplate.ContainerSpec.Command)
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Env) > 0 {
		err = d.Set("env", service.Spec.TaskTemplate.ContainerSpec.Env)
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Hosts) > 0 {
		err = d.Set("hosts", flattenServiceHosts(service.Spec.TaskTemplate.ContainerSpec.Hosts))
		if err != nil {
			return err
		}
	}
	if len(service.Endpoint.Spec.Mode) > 0 {
		err = d.Set("network_mode", service.Endpoint.Spec.Mode)
		if err != nil {
			return err
		}
	}
	if len(service.Spec.Networks) > 0 {
		err = d.Set("networks", flattenServiceNetworks(service.Spec.Networks))
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Mounts) > 0 {
		err = d.Set("mounts", flattenServiceMounts(service.Spec.TaskTemplate.ContainerSpec.Mounts))
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Configs) > 0 {
		err = d.Set("configs", flattenServiceConfigs(service.Spec.TaskTemplate.ContainerSpec.Configs))
		if err != nil {
			return err
		}
	}
	if len(service.Spec.TaskTemplate.ContainerSpec.Secrets) > 0 {
		err = d.Set("secrets", flattenServiceSecrets(service.Spec.TaskTemplate.ContainerSpec.Secrets))
		if err != nil {
			return err
		}
	}
	if len(service.Endpoint.Spec.Ports) > 0 {
		err = d.Set("ports", flattenServicePorts(service.Endpoint.Spec.Ports))
		if err != nil {
			return err
		}
	}
	// TOOD float64
	// if service.Spec.UpdateConfig != nil {
	// 	err = d.Set("update_config", flattenServiceUpdateOrRollbackConfig(service.Spec.UpdateConfig))
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if service.Spec.RollbackConfig != nil {
	// 	err = d.Set("rollback_config", flattenServiceUpdateOrRollbackConfig(service.Spec.RollbackConfig))
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// TOOD docker_service.foo: Invalid address to set: []string{"constraints"}
	if service.Spec.TaskTemplate.Placement != nil {
		err = d.Set("placement", flattenServicePlacement(service.Spec.TaskTemplate.Placement))
		if err != nil {
			return err
		}
	}

	if service.Spec.TaskTemplate.LogDriver != nil {
		err = d.Set("logging", flattenServiceLogging(service.Spec.TaskTemplate.LogDriver))
		if err != nil {
			return err
		}
	}

	if service.Spec.TaskTemplate.ContainerSpec.Healthcheck != nil {
		err = d.Set("healthcheck", flattenServiceHealthcheck(service.Spec.TaskTemplate.ContainerSpec.Healthcheck))
		if err != nil {
			return err
		}
	}

	if service.Spec.TaskTemplate.ContainerSpec.DNSConfig != nil {
		err = d.Set("dns_config", flattenServiceDNSConfig(service.Spec.TaskTemplate.ContainerSpec.DNSConfig))
		if err != nil {
			return err
		}
	}

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

	if v, ok := d.GetOk("update_config"); ok {
		updateOpts.UpdateConfig, _ = createUpdateOrRollbackConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("rollback_config"); ok {
		updateOpts.RollbackConfig, _ = createUpdateOrRollbackConfig(v.([]interface{}))
	}

	if err = client.UpdateService(d.Id(), updateOpts); err != nil {
		return err
	}

	if v, ok := d.GetOk("converge_config"); ok {
		convergeConfig := createConvergeConfig(v.([]interface{}))
		log.Printf("[INFO] Waiting for Service '%s' to be updated with timeout: %v", service.ID, convergeConfig.timeoutRaw)
		timeout, _ := time.ParseDuration(convergeConfig.timeoutRaw)
		stateConf := &resource.StateChangeConf{
			Pending:    serviceUpdatePendingStates,
			Target:     []string{"completed"},
			Refresh:    resourceDockerServiceUpdateRefreshFunc(service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      7 * time.Second,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForState()
		if err != nil {
			// the service will be deleted in case it cannot be converged
			if deleteErr := deleteService(service.ID, d, client); deleteErr != nil {
				return deleteErr
			}
			if strings.Contains(err.Error(), "timeout while waiting for state") {
				return &DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout}
			}
			return err
		}
	}

	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	if err := deleteService(d.Id(), d, client); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

/////////////////
// Helpers
/////////////////
// fetchDockerService fetches a service by its name or id
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

// deleteService deletes the service with the given id
func deleteService(serviceID string, d *schema.ResourceData, client *dc.Client) error {
	// get containerIDs of the running service because they do not exist after the service is deleted
	serviceContainerIds := make([]string, 0)
	if _, ok := d.GetOk("destroy_grace_seconds"); ok {
		filter := make(map[string][]string)
		filter["service"] = []string{d.Get("name").(string)}
		tasks, err := client.ListTasks(dc.ListTasksOptions{
			Filters: filter,
		})
		if err != nil {
			return err
		}
		for _, t := range tasks {
			task, _ := client.InspectTask(t.ID)
			log.Printf("[INFO] Found container ['%s'] for destroying: '%s'", task.Status.State, task.Status.ContainerStatus.ContainerID)
			if strings.TrimSpace(task.Status.ContainerStatus.ContainerID) != "" && task.Status.State != swarm.TaskStateShutdown {
				serviceContainerIds = append(serviceContainerIds, task.Status.ContainerStatus.ContainerID)
			}
		}
	}

	// delete the service
	log.Printf("[INFO] Deleting service: '%s'", serviceID)
	removeOpts := dc.RemoveServiceOptions{
		ID: serviceID,
	}

	if err := client.RemoveService(removeOpts); err != nil {
		if _, ok := err.(*dc.NoSuchService); ok {
			log.Printf("[WARN] Service (%s) not found, removing from state", serviceID)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error deleting service %s: %s", serviceID, err)
	}

	// destroy each container after a grace period if specified
	if v, ok := d.GetOk("destroy_grace_seconds"); ok {
		for _, containerID := range serviceContainerIds {
			timeout := v.(int)
			destroyGraceSeconds := time.Duration(timeout) * time.Second
			log.Printf("[INFO] Waiting for container: '%s' to exit: max %v", containerID, destroyGraceSeconds)
			ctx, cancel := context.WithTimeout(context.Background(), destroyGraceSeconds)
			defer cancel()
			exitCode, _ := client.WaitContainerWithContext(containerID, ctx)
			log.Printf("[INFO] Container exited with code [%v]: '%s'", exitCode, containerID)

			removeOpts := dc.RemoveContainerOptions{
				ID:            containerID,
				RemoveVolumes: true,
				Force:         true,
			}

			log.Printf("[INFO] Removing container: '%s'", containerID)
			if err := client.RemoveContainer(removeOpts); err != nil {
				if !(strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "is already in progress")) {
					return fmt.Errorf("Error deleting container %s: %s", containerID, err)
				}
			}
		}
	}

	return nil
}

//////// Convergers

// DidNotConvergeError is the error returned when a the service does not converge in
// the defined time
type DidNotConvergeError struct {
	ServiceID string
	Timeout   time.Duration
	Err       error
}

// Error the custom error if a service does not converge
func (err *DidNotConvergeError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "Service with ID (" + err.ServiceID + ") did not converge after " + err.Timeout.String()
}

// resourceDockerServiceCreateRefreshFunc refreshes the state of a service when it is created and needs to converge
func resourceDockerServiceCreateRefreshFunc(
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		ctx := context.Background()

		var updater progressUpdater

		if updater == nil {
			updater = &replicatedConsoleLogUpdater{}
		}

		filter := make(map[string][]string)
		filter["service"] = []string{serviceID}
		filter["desired-state"] = []string{"running"}

		getUpToDateTasks := func() ([]swarm.Task, error) {
			return client.ListTasks(dc.ListTasksOptions{
				Filters: filter,
				Context: ctx,
			})
		}
		var service *swarm.Service
		service, err := client.InspectService(serviceID)
		if err != nil {
			return nil, "", err
		}

		tasks, err := getUpToDateTasks()
		if err != nil {
			return nil, "", err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return nil, "", err
		}

		serviceCreateStatus, err := updater.update(service, tasks, activeNodes, false)
		if err != nil {
			return nil, "", err
		}

		if serviceCreateStatus {
			return service.ID, "running", nil
		}

		return service.ID, "creating", nil
	}
}

// resourceDockerServiceUpdateRefreshFunc refreshes the state of a service when it is updated and needs to converge
func resourceDockerServiceUpdateRefreshFunc(
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		ctx := context.Background()

		var (
			updater  progressUpdater
			rollback bool
		)

		if updater == nil {
			updater = &replicatedConsoleLogUpdater{}
		}
		rollback = false

		filter := make(map[string][]string)
		filter["service"] = []string{serviceID}
		filter["desired-state"] = []string{"running"}

		getUpToDateTasks := func() ([]swarm.Task, error) {
			return client.ListTasks(dc.ListTasksOptions{
				Filters: filter,
				Context: ctx,
			})
		}
		var service *swarm.Service
		service, err := client.InspectService(serviceID)
		if err != nil {
			return nil, "", err
		}

		if service.UpdateStatus != nil {
			switch service.UpdateStatus.State {
			case swarm.UpdateStateUpdating:
				rollback = false
			case swarm.UpdateStateCompleted:
				return service.ID, "completed", nil
			case swarm.UpdateStateRollbackStarted:
				rollback = true
			case swarm.UpdateStateRollbackCompleted:
				return nil, "", fmt.Errorf("service rollback completed: %s", service.UpdateStatus.Message)
			case swarm.UpdateStatePaused:
				return nil, "", fmt.Errorf("service update paused: %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackPaused:
				return nil, "", fmt.Errorf("service rollback paused: %s", service.UpdateStatus.Message)
			}
		}

		tasks, err := getUpToDateTasks()
		if err != nil {
			return nil, "", err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return nil, "", err
		}

		isUpdateCompleted, err := updater.update(service, tasks, activeNodes, rollback)
		if err != nil {
			return nil, "", err
		}

		if isUpdateCompleted {
			return service.ID, "completed", nil
		}

		return service.ID, "updating", nil
	}
}

// getActiveNodes gets the actives nodes withon a swarm
func getActiveNodes(ctx context.Context, client *dc.Client) (map[string]struct{}, error) {
	nodes, err := client.ListNodes(dc.ListNodesOptions{Context: ctx})
	if err != nil {
		return nil, err
	}

	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}

// progressUpdater interface for progressive task updates
type progressUpdater interface {
	update(service *swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error)
}

// replicatedConsoleLogUpdater console log updater for replicated services
type replicatedConsoleLogUpdater struct {
	// used for mapping slots to a contiguous space
	// this also causes progress bars to appear in order
	slotMap map[int]int

	initialized bool
	done        bool
}

// update is the concrete implementation of updating replicated services
func (u *replicatedConsoleLogUpdater) update(service *swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return false, fmt.Errorf("no replica count")
	}
	replicas := *service.Spec.Mode.Replicated.Replicas

	if !u.initialized {
		u.slotMap = make(map[int]int)
		u.initialized = true
	}

	// get the task for each slot. there can be multiple slots on one node
	tasksBySlot := u.tasksBySlot(tasks, activeNodes)

	// if a converged state is reached, check if is still converged.
	if u.done {
		for _, task := range tasksBySlot {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := uint64(0)

	// map the slots to keep track of their state individually
	for _, task := range tasksBySlot {
		mappedSlot := u.slotMap[task.Slot]
		if mappedSlot == 0 {
			mappedSlot = len(u.slotMap) + 1
			u.slotMap[task.Slot] = mappedSlot
		}

		// if a task is in the desired state count it as running
		if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
			running++
		}
	}

	// check if all tasks the same amount of tasks is running than replicas defined
	if !u.done {
		log.Printf("[INFO] ... progress: [%v/%v] - rollback: %v", running, replicas, rollback)
		if running == replicas {
			log.Printf("[INFO] DONE: all %v replicas running", running)
			u.done = true
		}
	}

	return running == replicas, nil
}

// tasksBySlot maps the tasks to slots on active nodes. There can be multiple slots on active nodes.
// A task is analogous to a “slot” where (on a node) the scheduler places a container.
func (u *replicatedConsoleLogUpdater) tasksBySlot(tasks []swarm.Task, activeNodes map[string]struct{}) map[int]swarm.Task {
	// if there are multiple tasks with the same slot number, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksBySlot[task.Slot]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}
			// if the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		// if the task is on a node and this node is active, then map this task to a slot
		if task.NodeID != "" {
			if _, nodeActive := activeNodes[task.NodeID]; !nodeActive {
				continue
			}
		}
		tasksBySlot[task.Slot] = task
	}

	return tasksBySlot
}

// terminalState determines if the given state is a terminal state
// meaninig 'higher' than running (see numberedStates)
func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

//////// Mappers
// createServiceSpec creates the service spec
func createServiceSpec(d *schema.ResourceData) (swarm.ServiceSpec, error) {

	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
		TaskTemplate: swarm.TaskSpec{},
	}

	if v, ok := d.GetOk("mode"); ok {
		serviceSpec.Mode = swarm.ServiceMode{}
		// because its a list
		if len(v.([]interface{})) > 0 {
			for _, rawMode := range v.([]interface{}) {
				// with a map
				rawMode := rawMode.(map[string]interface{})

				if rawReplicatedMode, ok := rawMode["replicated"]; ok {
					// with a set
					rawReplicatedModeSet := rawReplicatedMode.(*schema.Set)
					for _, rawReplicatedModeInt := range rawReplicatedModeSet.List() {
						// which is a map
						rawReplicatedModeMap := rawReplicatedModeInt.(map[string]interface{})
						serviceSpec.Mode.Replicated = &swarm.ReplicatedService{}
						if testReplicas, ok := rawReplicatedModeMap["replicas"]; ok {
							replicas := uint64(testReplicas.(int))
							serviceSpec.Mode.Replicated.Replicas = &replicas
						}
					}
				} else {
					if _, ok := rawMode["global"]; ok {
						serviceSpec.Mode.Global = &swarm.GlobalService{}
					}
				}
			}
		}
	}

	// == start Container Spec
	containerSpec := swarm.ContainerSpec{
		Image: d.Get("image").(string),
	}

	if v, ok := d.GetOk("hostname"); ok {
		containerSpec.Hostname = v.(string)
	}

	if v, ok := d.GetOk("stop_grace_period"); ok {
		parsed, _ := time.ParseDuration(v.(string))
		containerSpec.StopGracePeriod = &parsed
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
			if w, ok := rawMount["consistency"]; ok {
				mountInstance.Consistency = mount.Consistency(w.(string))
			}

			mountInstance.VolumeOptions = &mount.VolumeOptions{}
			if w, ok := rawMount["volume_labels"]; ok {
				mountInstance.VolumeOptions.Labels = mapTypeMapValsToString(w.(map[string]interface{}))
			}

			if mountType == mount.TypeBind {
				if w, ok := rawMount["bind_propagation"]; ok {
					mountInstance.BindOptions = &mount.BindOptions{
						Propagation: mount.Propagation(w.(string)),
					}
				}
			} else if mountType == mount.TypeVolume {
				if w, ok := rawMount["volume_no_copy"]; ok {
					mountInstance.VolumeOptions.NoCopy = w.(bool)
				}

				mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
				if w, ok := rawMount["volume_driver_name"]; ok {
					mountInstance.VolumeOptions.DriverConfig.Name = w.(string)
				}

				if w, ok := rawMount["volume_driver_options"]; ok {
					mountInstance.VolumeOptions.DriverConfig.Options = mapTypeMapValsToString(w.(map[string]interface{}))
				}
			} else if mountType == mount.TypeTmpfs {
				mountInstance.TmpfsOptions = &mount.TmpfsOptions{}

				if w, ok := rawMount["tmpfs_size_bytes"]; ok {
					mountInstance.TmpfsOptions.SizeBytes = w.(int64)
				}

				if w, ok := rawMount["tmpfs_mode"]; ok {
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
	if v, ok := d.GetOk("placement"); ok {
		placement := swarm.Placement{}
		for _, rawPlacement := range v.([]interface{}) {
			rawPlacement := rawPlacement.(map[string]interface{})
			if v, ok := rawPlacement["constraints"]; ok {
				placement.Constraints = stringSetToStringSlice(v.(*schema.Set))
			}

			if v, ok := rawPlacement["prefs"]; ok {
				placement.Preferences = stringSetToPlacementPrefs(v.(*schema.Set))
			}

			if v, ok := rawPlacement["platforms"]; ok {
				placement.Platforms = mapSetToPlacementPlatforms(v.(*schema.Set))
			}
		}
		serviceSpec.TaskTemplate.Placement = &placement
	}

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

	if v, ok := d.GetOk("dns_config"); ok {
		containerSpec.DNSConfig = &swarm.DNSConfig{}
		if len(v.([]interface{})) > 0 {
			for _, rawDNSConfig := range v.([]interface{}) {
				rawDNSConfig := rawDNSConfig.(map[string]interface{})
				if nameservers, ok := rawDNSConfig["nameservers"]; ok {
					containerSpec.DNSConfig.Nameservers = stringListToStringSlice(nameservers.([]interface{}))
				}
				if search, ok := rawDNSConfig["search"]; ok {
					containerSpec.DNSConfig.Search = stringListToStringSlice(search.([]interface{}))
				}
				if options, ok := rawDNSConfig["options"]; ok {
					containerSpec.DNSConfig.Options = stringListToStringSlice(options.([]interface{}))
				}
			}
		}
	}

	// == end TaskTemplate Spec

	return serviceSpec, nil
}

// createUpdateOrRollbackConfig create the configuration for and update or rollback
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
			log.Printf("[INFO] --> setting: %v", float32(v.(float64)))
			updateConfig.MaxFailureRatio = float32(v.(float64))
		}
		if v, ok := sc["order"]; ok {
			updateConfig.Order = v.(string)
		}
	}

	return &updateConfig, nil
}

// createConvergeConfig creates the configuration for converging
func createConvergeConfig(config []interface{}) *convergeConfig {
	plainConvergeConfig := &convergeConfig{}
	if len(config) > 0 {
		for _, rawConvergeConfig := range config {
			rawConvergeConfig := rawConvergeConfig.(map[string]interface{})
			if interval, ok := rawConvergeConfig["interval"]; ok {
				plainConvergeConfig.interval, _ = time.ParseDuration(interval.(string))
			}
			if monitor, ok := rawConvergeConfig["monitor"]; ok {
				plainConvergeConfig.monitor, _ = time.ParseDuration(monitor.(string))
			}
			if timeout, ok := rawConvergeConfig["timeout"]; ok {
				plainConvergeConfig.timeoutRaw, _ = timeout.(string)
				plainConvergeConfig.timeout, _ = time.ParseDuration(timeout.(string))
			}
		}
	}
	return plainConvergeConfig
}

// portSetToServicePorts maps a set of ports to portConfig
func portSetToServicePorts(ports *schema.Set) []swarm.PortConfig {
	retPortConfigs := []swarm.PortConfig{}

	for _, portInt := range ports.List() {
		port := portInt.(map[string]interface{})
		internal := port["internal"].(int)
		protocol := port["protocol"].(string)
		external := internal
		if externalPort, ok := port["external"]; ok {
			external = externalPort.(int)
		}

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

// authToServiceAuth maps the auth to AuthConfiguration
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

// fromRegistryAuth extract the desired AuthConfiguration for the given image
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

// stringSetToPlacementPrefs maps a string set to PlacementPreference
func stringSetToPlacementPrefs(stringSet *schema.Set) []swarm.PlacementPreference {
	ret := []swarm.PlacementPreference{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, swarm.PlacementPreference{
			Spread: &swarm.SpreadOver{
				SpreadDescriptor: envVal.(string),
			},
		})
	}
	return ret
}

// mapSetToPlacementPlatforms maps a string set to Platform
func mapSetToPlacementPlatforms(stringSet *schema.Set) []swarm.Platform {
	ret := []swarm.Platform{}
	if stringSet == nil {
		return ret
	}

	for _, rawPlatform := range stringSet.List() {
		rawPlatform := rawPlatform.(map[string]interface{})
		ret = append(ret, swarm.Platform{
			Architecture: rawPlatform["architecture"].(string),
			OS:           rawPlatform["os"].(string),
		})
	}

	return ret
}

//////// States

// numberedStates are ascending sorted states for docker tasks
// meaning they appear internally in this order in the statemachine
var (
	numberedStates = map[swarm.TaskState]int64{
		swarm.TaskStateNew:       1,
		swarm.TaskStateAllocated: 2,
		swarm.TaskStatePending:   3,
		swarm.TaskStateAssigned:  4,
		swarm.TaskStateAccepted:  5,
		swarm.TaskStatePreparing: 6,
		swarm.TaskStateReady:     7,
		swarm.TaskStateStarting:  8,
		swarm.TaskStateRunning:   9,

		// The following states are not actually shown in progress
		// output, but are used internally for ordering.
		swarm.TaskStateComplete: 10,
		swarm.TaskStateShutdown: 11,
		swarm.TaskStateFailed:   12,
		swarm.TaskStateRejected: 13,
	}

	longestState int
)

// serviceCreatePendingStates are the pending states for the creation of a service
var serviceCreatePendingStates = []string{
	"new",
	"allocated",
	"pending",
	"assigned",
	"accepted",
	"preparing",
	"ready",
	"starting",
	"creating",
	"paused",
}

// serviceUpdatePendingStates are the pending states for the update of a service
var serviceUpdatePendingStates = []string{
	"creating",
	"updating",
}
