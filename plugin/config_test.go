package plugin

import (
	"fmt"
	"os"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/stretchr/testify/assert"
)

func Test_parse(t *testing.T) {
	testCases := []struct {
		client         *hcloud.Client
		input          interface{}
		expectedOutput hcloudTargetConfig
		expectedError  error
		name           string
	}{
		{
			client: hcloud.NewClient(
				hcloud.WithToken(os.Getenv("HCLOUD_TOKEN")),
				hcloud.WithEndpoint(os.Getenv("HCLOUD_ENDPOINT")),
			),
			input: map[string]interface{}{
				"hcloud_networks":  []string{"mynet"},
				"hcloud_location":  "fsn1",
				"hcloud_user_data": "#!/bin/bash",
				"hcloud_ssh_keys":  "my-resource",
				"hcloud_group_id":  "test",
			},
			expectedOutput: hcloudTargetConfig{
				Location: &hcloud.Location{
					Name: "fsn1",
				},
				UserData: "#!/bin/bash",
				SSHKeys: []*hcloud.SSHKey{
					&hcloud.SSHKey{
						Name: "my-resource",
					},
				},
				GroupID: "test",
				Networks: []*hcloud.Network{
					&hcloud.Network{
						Name: "mynet",
					},
				},
				ServerType: &hcloud.ServerType{
					Name: "cx11",
				},
				Image: &hcloud.Image{
					Name: "ubuntu-20.04",
				},
				PublicNetEnableIPv4: true,
			},
			expectedError: nil,
			name:          "successful network parse",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actualOutput hcloudTargetConfig
			actualError := parse(tc.client, tc.input, &actualOutput)
			assert.NotZero(t, actualOutput.Location.ID, fmt.Sprintf("Location: %s", tc.name))
			assert.NotZero(t, actualOutput.SSHKeys[0].ID, fmt.Sprintf("SSHKey: %s", tc.name))
			assert.Equal(t, tc.expectedError, actualError, tc.name)
		})
	}
}
