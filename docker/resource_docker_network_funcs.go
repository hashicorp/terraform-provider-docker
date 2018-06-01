package docker

import (
	"fmt"

	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func resourceDockerNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	createOpts := types.NetworkCreate{}
	if v, ok := d.GetOk("check_duplicate"); ok {
		createOpts.CheckDuplicate = v.(bool)
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("options"); ok {
		createOpts.Options = mapTypeMapValsToString(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("internal"); ok {
		createOpts.Internal = v.(bool)
	}

	ipamOpts := &network.IPAM{}
	ipamOptsSet := false
	if v, ok := d.GetOk("ipam_driver"); ok {
		ipamOpts.Driver = v.(string)
		ipamOptsSet = true
	}
	if v, ok := d.GetOk("ipam_config"); ok {
		ipamOpts.Config = ipamConfigSetToIpamConfigs(v.(*schema.Set))
		ipamOptsSet = true
	}

	if ipamOptsSet {
		createOpts.IPAM = ipamOpts
	}

	var err error
	var retNetwork types.NetworkCreateResponse
	if retNetwork, err = client.NetworkCreate(context.Background(), d.Get("name").(string), createOpts); err != nil {
		return fmt.Errorf("Unable to create network: %s", err)
	}

	d.SetId(retNetwork.ID)

	return resourceDockerNetworkRead(d, meta)
}

func resourceDockerNetworkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	var err error
	var retNetwork types.NetworkResource
	if retNetwork, err = client.NetworkInspect(context.Background(), d.Id(), types.NetworkInspectOptions{}); err != nil {
		log.Printf("[WARN] Network (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("scope", retNetwork.Scope)
	d.Set("driver", retNetwork.Driver)
	d.Set("options", retNetwork.Options)
	d.Set("internal", retNetwork.Internal)

	return nil
}

func resourceDockerNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	if err := client.NetworkRemove(context.Background(), d.Id()); err != nil {
		return fmt.Errorf("Error deleting network %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func ipamConfigSetToIpamConfigs(ipamConfigSet *schema.Set) []network.IPAMConfig {
	ipamConfigs := make([]network.IPAMConfig, ipamConfigSet.Len())

	for i, ipamConfigInt := range ipamConfigSet.List() {
		ipamConfigRaw := ipamConfigInt.(map[string]interface{})

		ipamConfig := network.IPAMConfig{}
		ipamConfig.Subnet = ipamConfigRaw["subnet"].(string)
		ipamConfig.IPRange = ipamConfigRaw["ip_range"].(string)
		ipamConfig.Gateway = ipamConfigRaw["gateway"].(string)

		auxAddressRaw := ipamConfigRaw["aux_address"].(map[string]interface{})
		ipamConfig.AuxAddress = make(map[string]string, len(auxAddressRaw))
		for k, v := range auxAddressRaw {
			ipamConfig.AuxAddress[k] = v.(string)
		}

		ipamConfigs[i] = ipamConfig
	}

	return ipamConfigs
}
