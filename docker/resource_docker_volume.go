package docker

import (
	"fmt"
	"log"
	"strings"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerVolumeCreate,
		Read:   resourceDockerVolumeRead,
		Delete: resourceDockerVolumeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"mountpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDockerVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	createOpts := dc.CreateVolumeOptions{}
	if v, ok := d.GetOk("name"); ok {
		createOpts.Name = v.(string)
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("driver_opts"); ok {
		createOpts.DriverOpts = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	var err error
	var retVolume *dc.Volume
	if retVolume, err = client.CreateVolume(createOpts); err != nil {
		return fmt.Errorf("Unable to create volume: %s", err)
	}
	if retVolume == nil {
		return fmt.Errorf("Returned volume is nil")
	}

	d.SetId(retVolume.Name)
	d.Set("name", retVolume.Name)
	d.Set("driver", retVolume.Driver)
	d.Set("mountpoint", retVolume.Mountpoint)

	return nil
}

func resourceDockerVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	var err error
	var retVolume *dc.Volume
	if retVolume, err = client.InspectVolume(d.Id()); err != nil && err != dc.ErrNoSuchVolume {
		return fmt.Errorf("Unable to inspect volume: %s", err)
	}
	if retVolume == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", retVolume.Name)
	d.Set("driver", retVolume.Driver)
	d.Set("mountpoint", retVolume.Mountpoint)

	return nil
}

func resourceDockerVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	// TODO catch error if removal is already in progress + fix with statefunc
	if err := client.RemoveVolume(d.Id()); err != nil && err != dc.ErrNoSuchVolume {
		if err == dc.ErrVolumeInUse {
			loops := 20
			sleepTime := 1000 * time.Millisecond
			for i := loops; i > 0; i-- {
				if err = client.RemoveVolume(d.Id()); err != nil {
					if err == dc.ErrVolumeInUse {
						log.Printf("[INFO] Volume remove loop: %d of %d due to error: %s", loops-i+1, loops, err)
						time.Sleep(sleepTime)
						continue
					}
					if err == dc.ErrNoSuchVolume {
						log.Printf("[INFO] Volume successfully removed")
						d.SetId("")
						return nil
					}
					if !strings.Contains(err.Error(), "is already in progress") {
						// if it's not in use any more (so it's deleted successfully) and another error occurred
						return fmt.Errorf("Error deleting volume %s: %s", d.Id(), err)
					}
				}
			}
			return fmt.Errorf("Error deleting volume %s: %s after %d tries", d.Id(), err, loops)
		}
	}

	d.SetId("")
	return nil
}
