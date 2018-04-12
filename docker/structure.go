package docker

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

func flattenServiceSpec(in swarm.ServiceSpec) []interface{} {
	att := make(map[string]interface{})

	att["name"] = in.Name
	att["task_template"] = flattenTaskTemplate(in.TaskTemplate)
	att["mode"] = "TODO"
	if in.UpdateConfig != nil {
		att["update_config"] = "TODO"
	}
	if in.RollbackConfig != nil {
		att["rollback_config"] = "TODO"
	}
	if in.EndpointSpec != nil {
		att["endpoint_config"] = "TODO"
	}

	return []interface{}{att}
}

// start TASK TEMPLATE
// flattenTaskTemplate
func flattenTaskTemplate(in swarm.TaskSpec) map[string]interface{} {
	m := make(map[string]interface{})
	if in.ContainerSpec != nil {
		m["container_spec"] = flattenContainerSpec(in.ContainerSpec)
	}
	if in.PluginSpec != nil {
		m["plugin_spec"] = "TODO"
	}
	if in.Resources != nil {
		m["resources"] = "TODO"
	}
	if in.RestartPolicy != nil {
		m["restart_policy"] = "TODO"
	}
	if in.Placement != nil {
		m["placement"] = "TODO"
	}
	if len(in.Networks) > 0 {
		m["networks"] = "TODO"
	}
	if in.LogDriver != nil {
		m["log_driver"] = "TODO"
	}
	m["force_update"] = in.ForceUpdate
	m["runtime"] = in.Runtime
	return m
}
func flattenContainerSpec(in *swarm.ContainerSpec) map[string]interface{} {
	m := make(map[string]interface{})
	m["image"] = in.Image
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
	if len(in.Dir) > 0 {
		m["dir"] = in.Dir
	}
	if len(in.User) > 0 {
		m["user"] = in.User
	}
	if len(in.Groups) > 0 {
		m["groups"] = in.Groups
	}
	if in.Privileges != nil {
		m["privileges"] = flattenPrivileges(in.Privileges)
	}
	if len(in.StopSignal) > 0 {
		m["stopsignal"] = in.StopSignal
	}
	m["tty"] = in.TTY
	m["openstdin"] = in.OpenStdin
	m["readonly"] = in.OpenStdin
	if len(in.Mounts) > 0 {
		m["mounts"] = flattenMounts(in.Mounts)
	}
	if in.StopGracePeriod != nil {
		m["stop_grace_period"] = in.StopGracePeriod
	}
	if in.Healthcheck != nil {
		m["healthcheck"] = flattenHealthcheck(in.Healthcheck)
	}
	if len(in.Hosts) > 0 {
		m["hosts"] = in.Hosts
	}
	if in.DNSConfig != nil {
		m["dns_config"] = flattenDNSConfig(in.DNSConfig)
	}
	if len(in.Secrets) > 0 {
		m["secrets"] = flattenSecrets(in.Secrets)
	}
	if len(in.Configs) > 0 {
		m["configs"] = flattenConfigs(in.Configs)
	}
	m["isolation"] = in.Isolation // is a string: type Isolation
	return m
}

func flattenPrivileges(in *swarm.Privileges) map[string]interface{} {
	m := make(map[string]interface{})

	if in.CredentialSpec != nil {
		m["credential_spec"] = in.CredentialSpec // is a struct of strings
	}
	if in.CredentialSpec != nil {
		m["se_linux_context"] = in.SELinuxContext // is a struct of strings
	}

	return m
}
func flattenMounts(in []mount.Mount) []interface{} {
	m := make(map[string]interface{})

	m["TODO"] = "TODO"

	return []interface{}{m}
}
func flattenHealthcheck(in *container.HealthConfig) map[string]interface{} {
	m := make(map[string]interface{})

	m["TODO"] = "TODO"

	return m
}
func flattenDNSConfig(in *swarm.DNSConfig) map[string]interface{} {
	m := make(map[string]interface{})

	m["TODO"] = "TODO"

	return m
}
func flattenSecrets(in []*swarm.SecretReference) map[string]interface{} {
	m := make(map[string]interface{})

	m["TODO"] = "TODO"

	return m
}
func flattenConfigs(in []*swarm.ConfigReference) map[string]interface{} {
	m := make(map[string]interface{})

	m["TODO"] = "TODO"

	return m
}

// end TASK TEMPLATE

// start ENDPOINT
// flattenServiceEndpoint
func flattenServiceEndpoint(in swarm.Endpoint) []interface{} {
	att := make(map[string]interface{})

	att["spec"] = "TODO"
	att["ports"] = "TODO"
	att["virtual_ips"] = "TODO"

	return []interface{}{att}
}

// end ENDPOINT
