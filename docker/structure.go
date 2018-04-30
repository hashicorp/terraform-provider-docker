package docker

import (
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform/helper/schema"
)

func flattenTaskSpec(in swarm.TaskSpec) []interface{} {
	m := make(map[string]interface{})
	if in.ContainerSpec != nil {
		m["container_spec"] = flattenContainerSpec(in.ContainerSpec)
	}
	if in.Resources != nil {
		m["resources"] = flattenTaskResources(in.Resources)
	}
	if in.RestartPolicy != nil {
		m["restart_policy"] = flattenTaskRestartPolicy(in.RestartPolicy)
	}
	if in.Placement != nil {
		m["placements"] = flattenTaskPlacement(in.Placement)
	}
	if in.ForceUpdate >= 0 {
		m["force_update"] = in.ForceUpdate
	}
	if len(in.Runtime) > 0 {
		m["runtime"] = in.Runtime
	}
	if len(in.Networks) > 0 {
		m["networks"] = flattenTaskNetworks(in.Networks)
	}
	if in.LogDriver != nil {
		m["log_driver"] = flattenTaskLogDriver(in.LogDriver)
	}

	return []interface{}{m}
}

func flattenServiceMode(in swarm.ServiceMode) []interface{} {
	m := make(map[string]interface{})
	if in.Replicated != nil {
		m["replicated"] = flattenReplicated(in.Replicated)
	}
	if in.Global != nil {
		m["global"] = true
	} else {
		m["global"] = false
	}

	return []interface{}{m}
}

func flattenReplicated(in *swarm.ReplicatedService) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	if in != nil {
		if in.Replicas != nil {
			replicas := int(*in.Replicas)
			m["replicas"] = replicas
		}
	}
	out = append(out, m)
	return out
}

func flattenServiceUpdateOrRollbackConfig(in *swarm.UpdateConfig) []interface{} {
	var out = make([]interface{}, 0, 0)
	if in == nil {
		return out
	}

	m := make(map[string]interface{})
	m["parallelism"] = in.Parallelism
	m["delay"] = shortDur(in.Delay)
	m["failure_action"] = in.FailureAction
	m["monitor"] = shortDur(in.Monitor)
	m["max_failure_ratio"] = strconv.FormatFloat(float64(in.MaxFailureRatio), 'f', 1, 64)
	m["order"] = in.Order
	out = append(out, m)
	return out
}

func flattenServiceEndpointSpec(in swarm.EndpointSpec) []interface{} {
	// FIXME
	// // endpoint_mode is only present if Ports are set
	// // https://docs.docker.com/network/overlay/#bypass-the-routing-mesh-for-a-swarm-service
	// if len(service.Endpoint.Spec.Ports) > 0 {
	// 	if err = d.Set("endpoint_mode", service.Endpoint.Spec.Mode); err != nil {
	// 		log.Printf("[WARN] failed to set endpoint_mode from API: %s", err)
	// 	}
	// }

	m := make(map[string]interface{})
	if len(in.Mode) > 0 {
		m["mode"] = in.Mode
	}
	if len(in.Ports) > 0 {
		m["ports"] = flattenServicePorts(in.Ports)
	}

	return []interface{}{m}
}

///// start TaskSpec
func flattenContainerSpec(in *swarm.ContainerSpec) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	if len(in.Image) > 0 {
		m["image"] = in.Image
	}
	if len(in.Labels) > 0 {
		m["labels"] = in.Labels
	}
	if len(in.Command) > 0 {
		m["command"] = in.Command
	}
	if len(in.Args) > 0 {
		m["args"] = in.Args
	}
	if len(in.Hostname) > 0 {
		m["hostname"] = in.Hostname
	}
	if len(in.Env) > 0 {
		m["env"] = in.Env
	}
	if len(in.User) > 0 {
		m["user"] = in.User
	}
	if len(in.Groups) > 0 {
		m["groups"] = in.Groups
	}
	if in.Privileges != nil {
		// m["privileges"] = flattenPrivileges(in.Privileges)
	}
	if in.ReadOnly {
		m["read_only"] = in.ReadOnly
	}
	if len(in.Mounts) > 0 {
		m["mounts"] = flattenServiceMounts(in.Mounts)
	}
	if len(in.StopSignal) > 0 {
		m["stop_signal"] = in.StopSignal
	}
	if in.StopGracePeriod != nil {
		m["stop_signal"] = shortDur(*in.StopGracePeriod)
	}
	if in.Healthcheck != nil {
		m["healthcheck"] = flattenServiceHealthcheck(in.Healthcheck)
	}
	if len(in.Hosts) > 0 {
		m["hosts"] = flattenServiceHosts(in.Hosts)
	}
	if in.DNSConfig != nil {
		m["dns_config"] = flattenServiceDNSConfig(in.DNSConfig)
	}
	if len(in.Secrets) > 0 {
		m["secrets"] = flattenServiceSecrets(in.Secrets)
	}
	if len(in.Configs) > 0 {
		m["configs"] = flattenServiceConfigs(in.Configs)
	}
	out = append(out, m)
	return out
}

func flattenServiceMounts(in []mount.Mount) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["target"] = v.Target
		m["source"] = v.Source
		m["type"] = v.Type
		if len(v.Consistency) > 0 {
			m["consistency"] = v.Consistency
		}
		m["read_only"] = v.ReadOnly
		if v.BindOptions != nil {
			m["bind_propagation"] = v.BindOptions.Propagation
		}
		if v.VolumeOptions != nil {
			m["volume_no_copy"] = v.VolumeOptions.NoCopy
			m["volume_labels"] = v.VolumeOptions.Labels
			if v.VolumeOptions.DriverConfig != nil {
				m["volume_driver_name"] = v.VolumeOptions.DriverConfig.Name
				m["volume_driver_options"] = v.VolumeOptions.DriverConfig.Options
			}
		}
		if v.TmpfsOptions != nil {
			m["tmpfs_size_bytes"] = v.TmpfsOptions.SizeBytes
			m["tmpfs_mode"] = v.TmpfsOptions.Mode.Perm
		}
		out[i] = m
	}
	return out
}

func flattenServiceHealthcheck(in *container.HealthConfig) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	if len(in.Test) > 0 {
		m["test"] = in.Test
	}
	m["interval"] = shortDur(in.Interval)
	m["timeout"] = shortDur(in.Timeout)
	m["start_period"] = shortDur(in.StartPeriod)
	m["retries"] = in.Retries
	out[0] = m
	return out
}

func flattenServiceHosts(in []string) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		split := strings.Split(v, ":")
		m["host"] = split[0]
		m["ip"] = split[1]
		out[i] = m
	}
	return out
}

func flattenServiceDNSConfig(in *swarm.DNSConfig) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	if len(in.Nameservers) > 0 {
		m["nameservers"] = in.Nameservers
	}
	if len(in.Search) > 0 {
		m["search"] = in.Search
	}
	if len(in.Options) > 0 {
		m["options"] = in.Options
	}
	out[0] = m
	return out
}

func flattenServiceSecrets(in []*swarm.SecretReference) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["secret_id"] = v.SecretID
		if len(v.SecretName) > 0 {
			m["secret_name"] = v.SecretName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
		}
		out[i] = m
	}
	return out
}

func flattenServiceConfigs(in []*swarm.ConfigReference) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["config_id"] = v.ConfigID
		if len(v.ConfigName) > 0 {
			m["config_name"] = v.ConfigName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
		}
		out[i] = m
	}
	return out
}

func flattenTaskResources(in *swarm.ResourceRequirements) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	//
	out = append(out, m)
	return out
}

func flattenTaskRestartPolicy(in *swarm.RestartPolicy) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	//
	out = append(out, m)
	return out
}

func flattenTaskPlacement(in *swarm.Placement) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	if len(in.Constraints) > 0 {
		m["constraints"] = newStringSet(schema.HashString, in.Constraints)
	}
	if len(in.Preferences) > 0 {
		m["prefs"] = flattenPlacementPrefs(in.Preferences)
	}
	if len(in.Platforms) > 0 {
		m["platforms"] = flattenPlacementPlatforms(in.Platforms)
	}
	out[0] = m
	return out
}

func flattenPlacementPrefs(in []swarm.PlacementPreference) *schema.Set {
	if in == nil || len(in) == 0 {
		return schema.NewSet(schema.HashString, make([]interface{}, 0, 0))
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v.Spread.SpreadDescriptor
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenPlacementPlatforms(in []swarm.Platform) *schema.Set {
	if in == nil || len(in) == 0 {
		return schema.NewSet(schema.HashString, make([]interface{}, 0, 0))
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["architecture"] = v.Architecture
		m["os"] = v.OS
		out[i] = m
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenTaskNetworks(in []swarm.NetworkAttachmentConfig) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v.Target
	}
	return out
}

func flattenTaskLogDriver(in *swarm.Driver) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	m["driver_name"] = in.Name
	if len(in.Options) > 0 {
		m["options"] = in.Options
	}
	out[0] = m
	return out
}

///// end TaskSpec
///// start EndpointSpec
func flattenServicePorts(in []swarm.PortConfig) []interface{} {
	if in == nil || len(in) == 0 {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["internal"] = int(v.TargetPort)
		if v.PublishedPort > 0 {
			m["external"] = v.PublishedPort
		}
		m["publish_mode"] = v.PublishMode
		m["protocol"] = v.Protocol
		out[i] = m
	}
	return out
}

///// end EndpointSpec

// HELPERS
func shortDur(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func newStringSet(f schema.SchemaSetFunc, in []string) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}
