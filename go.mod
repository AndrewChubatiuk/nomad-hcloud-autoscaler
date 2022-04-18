module github.com/AndrewChubatiuk/nomad-hcloud-autoscaler

go 1.16

require (
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/creasty/defaults v1.6.0
	github.com/fatih/color v1.13.0 // indirect
	github.com/go-playground/locales v0.14.0
	github.com/go-playground/universal-translator v0.18.0
	github.com/go-playground/validator/v10 v10.10.1
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/consul/api v1.12.0 // indirect
	github.com/hashicorp/go-hclog v1.2.0
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/hcl/v2 v2.11.1 // indirect
	github.com/hashicorp/nomad-autoscaler v0.3.6
	github.com/hashicorp/nomad/api v0.0.0-20220412123539-86ca8f7e736e
	github.com/hashicorp/serf v0.9.7 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/hetznercloud/hcloud-go v1.33.1
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/oklog/run v1.1.0 // indirect
	github.com/prometheus/common v0.33.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/zclconf/go-cty v1.10.0 // indirect
	golang.org/x/net v0.0.0-20220412020605-290c469a71a5 // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
)

replace github.com/hetznercloud/hcloud-go v1.33.1 => github.com/AndrewChubatiuk/hcloud-go v1.33.2
