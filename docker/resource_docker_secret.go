package docker

import (
	"encoding/base64"
	"log"

	"github.com/docker/docker/api/types/swarm"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerSecretCreate,
		Read:   resourceDockerSecretRead,
		Update: resourceDockerSecretUpdate,
		Delete: resourceDockerSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"data": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateStringIsBase64Encoded(),
			},

			// Workaround until https://github.com/moby/moby/issues/35803 is fixed
			"updateable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
		},
	}
}

func resourceDockerSecretCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	// is validated on resource creation
	data, _ := base64.StdEncoding.DecodeString(d.Get("data").(string))

	createSecretOpts := dc.CreateSecretOptions{
		SecretSpec: swarm.SecretSpec{
			Annotations: swarm.Annotations{
				Name: d.Get("name").(string),
			},
			Data: data,
		},
	}

	secret, err := client.CreateSecret(createSecretOpts)
	if err != nil {
		return err
	}

	d.SetId(secret.ID)

	return resourceDockerSecretRead(d, meta)
}

func resourceDockerSecretRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	secret, err := client.InspectSecret(d.Id())

	if err != nil {
		if _, ok := err.(*dc.NoSuchSecret); ok {
			log.Printf("[WARN] Secret (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	d.SetId(secret.ID)
	return nil
}

func resourceDockerSecretUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: atm only the labels of a config can be updated. not the data
	// Wait for https://github.com/moby/moby/issues/35803
	client := meta.(*ProviderConfig).DockerClient
	data, err := base64.StdEncoding.DecodeString(d.Get("data").(string))

	err = client.UpdateSecret(d.Id(), dc.UpdateSecretOptions{
		SecretSpec: swarm.SecretSpec{
			Data: data,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func resourceDockerSecretDelete(d *schema.ResourceData, meta interface{}) error {
	// HACK configs simply cannot be deleted to have an update mechanism
	// Wait for https://github.com/moby/moby/issues/35803
	isUpdateable := d.Get("updateable").(bool)
	if !isUpdateable {
		client := meta.(*ProviderConfig).DockerClient
		err := client.RemoveSecret(dc.RemoveSecretOptions{
			ID: d.Id(),
		})

		if err != nil {
			return err
		}
	}

	d.SetId("")
	return nil
}
