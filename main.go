package main

import (
	hcloud "github.com/AndrewChubatiuk/nomad-hcloud-autoscaler/plugin"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
)

func main() {
	plugins.Serve(factory)
}

// factory returns a new instance of the AWS ASG plugin.
func factory(log hclog.Logger) interface{} {
	return hcloud.NewHCloudServerPlugin(log)
}
