package forward

import "fmt"

// checkConfig checks the config if it is feasible
func checkConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("Config cannot be nil")
	}

	if len(config.JumpHostConfigs) > 1 { //TODO atm only one jump host is supported
		return fmt.Errorf("Only 1 jump host is supported atm")
	}
	for _, jumpConfig := range config.JumpHostConfigs {
		if err := checkSSHConfig(jumpConfig); err != nil {
			return err
		}
	}
	if err := checkSSHConfig(config.EndHostConfig); err != nil {
		return err
	}
	if config.LocalAddress == "" || config.RemoteAddress == "" {
		return fmt.Errorf("LocalAddress and RemoteAddress have to be set")
	}

	return nil
}

// checkSSSConfig checks the ssh config for feasibility
func checkSSHConfig(sshConfig *SSHConfig) error {
	if sshConfig == nil {
		return fmt.Errorf("SSHConfig cannot be nil")
	}
	if sshConfig.User == "" {
		return fmt.Errorf("User cannot be empty")
	}
	if sshConfig.Address == "" {
		return fmt.Errorf("Address cannot be empty")
	}
	if sshConfig.PrivateKeyFile == "" && sshConfig.Password == "" {
		return fmt.Errorf("Either PrivateKeyFile or Password has to be set")
	}

	return nil
}
