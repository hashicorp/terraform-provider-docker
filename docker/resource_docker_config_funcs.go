package docker

import (
	"encoding/base64"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerConfigCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	// is validated on resource creation
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	createConfigOpts := dc.CreateConfigOptions{
		ConfigSpec: swarm.ConfigSpec{
			Annotations: swarm.Annotations{
				Name: d.Get("name").(string),
			},
			Data: data,
		},
	}

	config, err := client.CreateConfig(createConfigOpts)
	if err != nil {
		return err
	}
	d.SetId(config.ID)

	return nil
}

func resourceDockerConfigRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	config, err := client.InspectConfig(d.Id())

	if err != nil || config == nil {
		d.SetId("")
	}

	return nil
}

func resourceDockerConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: atm only the labels of a config can be updated. not the data
	// Wait for https://github.com/moby/moby/issues/35803
	client := meta.(*ProviderConfig).DockerClient
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	err := client.UpdateConfig(d.Id(), dc.UpdateConfigOptions{
		ConfigSpec: swarm.ConfigSpec{
			Data: data,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func resourceDockerConfigDelete(d *schema.ResourceData, meta interface{}) error {
	// HACK configs simply cannot be deleted to have an update mechanism
	// Wait for https://github.com/moby/moby/issues/35803
	isUpdateable := d.Get("updateable").(bool)
	if !isUpdateable {
		client := meta.(*ProviderConfig).DockerClient
		err := client.RemoveConfig(dc.RemoveConfigOptions{
			ID: d.Id(),
		})
		if err != nil {
			return err
		}
	}

	d.SetId("")
	return nil
}
