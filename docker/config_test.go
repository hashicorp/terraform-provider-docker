package docker

import "testing"

func TestParseForwardConfigEndHostOnly(t *testing.T) {
	// prepare
	forwardConfigWrapper := make([]interface{}, 0)
	forwardConfig := make(map[string]interface{})

	forwardConfig["end_host"] = "10.0.0.1:22"
	forwardConfig["end_host_user"] = "endhostuser"
	forwardConfig["end_host_private_key_file"] = "/Users/abc/.ssh/id_rsa_end_host"
	forwardConfig["local_address"] = "localhost:2376"
	forwardConfig["remote_address"] = "localhost:2377"

	forwardConfigWrapper = append(forwardConfigWrapper, forwardConfig)

	// go
	parsedForwardConfig, _ := parseForwardConfig(forwardConfigWrapper)

	// validate
	if len(parsedForwardConfig.JumpHostConfigs) != 0 {
		got := len(parsedForwardConfig.JumpHostConfigs)
		t.Fatalf("Wanted 0 jump host configs but got %d", got)
	}

	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.Address, "10.0.0.1:22")
	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.User, "endhostuser")
	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.PrivateKeyFile, "/Users/abc/.ssh/id_rsa_end_host")
	expectPropertyToBe(t, parsedForwardConfig.LocalAddress, "localhost:2376")
	expectPropertyToBe(t, parsedForwardConfig.RemoteAddress, "localhost:2377")
}
func TestParseForwardConfigWithBastionHost(t *testing.T) {
	// prepare
	forwardConfigWrapper := make([]interface{}, 0)
	forwardConfig := make(map[string]interface{})

	forwardConfig["bastion_host"] = "11.0.0.1:22"
	forwardConfig["bastion_host_user"] = "bastionhostuser"
	forwardConfig["bastion_host_private_key_file"] = "/Users/abc/.ssh/id_rsa_bastion_host"
	forwardConfig["end_host"] = "10.0.0.1:22"
	forwardConfig["end_host_user"] = "endhostuser"
	forwardConfig["end_host_private_key_file"] = "/Users/abc/.ssh/id_rsa_end_host"
	forwardConfig["local_address"] = "localhost:2376"
	forwardConfig["remote_address"] = "localhost:2377"

	forwardConfigWrapper = append(forwardConfigWrapper, forwardConfig)

	// go
	parsedForwardConfig, _ := parseForwardConfig(forwardConfigWrapper)

	// validate
	if len(parsedForwardConfig.JumpHostConfigs) != 1 {
		got := len(parsedForwardConfig.JumpHostConfigs)
		t.Fatalf("Wanted 1 jump host configs but got %d", got)
	}

	expectPropertyToBe(t, parsedForwardConfig.JumpHostConfigs[0].Address, "11.0.0.1:22")
	expectPropertyToBe(t, parsedForwardConfig.JumpHostConfigs[0].User, "bastionhostuser")
	expectPropertyToBe(t, parsedForwardConfig.JumpHostConfigs[0].PrivateKeyFile, "/Users/abc/.ssh/id_rsa_bastion_host")
	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.Address, "10.0.0.1:22")
	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.User, "endhostuser")
	expectPropertyToBe(t, parsedForwardConfig.EndHostConfig.PrivateKeyFile, "/Users/abc/.ssh/id_rsa_end_host")
	expectPropertyToBe(t, parsedForwardConfig.LocalAddress, "localhost:2376")
	expectPropertyToBe(t, parsedForwardConfig.RemoteAddress, "localhost:2377")
}

///////////
// Helper
///////////
func expectPropertyToBe(t *testing.T, wanted, got string) {
	if wanted != got {
		t.Fatalf("Wanted property to be \n'%s' but got \n'%s'", wanted, got)
	}
}
