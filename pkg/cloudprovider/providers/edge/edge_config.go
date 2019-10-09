/*
Copyright 2019 Midokura

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package edge

import (
	"bufio"
	"io"
	"os"

	"k8s.io/klog"
	//gcfg "gopkg.in/gcfg.v1"
)

// Config is used to read and store information from the cloud configuration file
type Config struct {
	Username string
	Password string
}

// ReadConfig reads values from environment variables and the cloud.conf, prioritizing cloud-config
func ReadConfig(config io.Reader) (Config, error) {
	klog.Infof("ReadConfig begin")

	cfg := configFromEnv()
	klog.V(5).Infof("Config loaded from the environment variables:")
	logCfg(cfg)

	//err := gcfg.FatalOnly(gcfg.ReadInto(&cfg, config))
	klog.V(5).Infof("Config file contents:")
	scanner := bufio.NewScanner(config)
	for scanner.Scan() {
		klog.V(5).Infof("  %s", scanner.Text())
	}

	klog.V(5).Infof("Config after adding the config file:")
	logCfg(cfg)

	return cfg, nil //err
}

func configFromEnv() Config {
	var cfg Config

	cfg.Username = os.Getenv("MIDOKURA_USERNAME")
	cfg.Password = os.Getenv("MIDOKURA_PASSWORD")

	return cfg
}

func logCfg(cfg Config) {
	klog.V(5).Infof("Username: %s", cfg.Username)
	klog.V(5).Infof("Password: %s", cfg.Password)
}
