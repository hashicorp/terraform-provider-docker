package docker

import (
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

func flattenServiceMode(in swarm.ServiceMode) []interface{} { // List
	att := make(map[string]interface{}) // map

	if in.Replicated != nil {
		att["replicated"] = flattenReplicated(in.Replicated)
	}
	if in.Global != nil {
		att["global"] = true
	} else {
		att["global"] = false
	}

	return []interface{}{att}
}

func flattenReplicated(in *swarm.ReplicatedService) []interface{} {
	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	iPtr := in.Replicas
	i := *in.Replicas
	log.Printf("[INFO] ---> %v -- %v", iPtr, i)
	m["replicas"] = int(i)
	out[0] = m
	return out
}

func flattenServiceHosts(in []string) []interface{} {
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

func flattenServiceNetworks(in []swarm.NetworkAttachmentConfig) []interface{} {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v.Target
	}
	return out
}

func flattenServiceMounts(in []mount.Mount) []interface{} {
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

func flattenServiceConfigs(in []*swarm.ConfigReference) []interface{} {
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

func flattenServiceSecrets(in []*swarm.SecretReference) []interface{} {
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

func flattenServicePorts(in []swarm.PortConfig) []interface{} {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["internal"] = int(v.TargetPort)
		if v.PublishedPort > 0 {
			m["external"] = v.PublishedPort
		}
		m["mode"] = v.PublishMode
		m["protocol"] = v.Protocol
		out[i] = m
	}
	return out
}

func flattenServiceUpdateOrRollbackConfig(in *swarm.UpdateConfig) []interface{} {
	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	m["parallelism"] = in.Parallelism
	m["delay"] = shortDur(in.Delay)
	m["failure_action"] = in.FailureAction
	m["monitor"] = shortDur(in.Monitor)
	log.Printf("[INFO] ----> plain :  %v", in.MaxFailureRatio)
	// log.Printf("[INFO] ----> casted:  %v", float32(float64(in.MaxFailureRatio)))
	m["max_failure_ratio"] = in.MaxFailureRatio
	// val, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", in.MaxFailureRatio), 32)
	// m["max_failure_ratio"] = float32(float64(in.MaxFailureRatio))
	// m["max_failure_ratio"], _ = strconv.ParseFloat(fmt.Sprintf("%.1f", in.MaxFailureRatio), 32)
	m["order"] = in.Order
	out[0] = m
	return out
}

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

func flattenVolumenLabels(in swarm.TaskSpec) map[string]interface{} {
	m := make(map[string]interface{})
	m["TODO"] = "TODO"
	return m
}
