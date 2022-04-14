package plugin

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/nomad/api"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

const (
	defaultRetryInterval   = 60 * time.Second
	defaultRetryLimit      = 5
	defaultPerPage         = 50
	defaultRandomSuffixLen = 10
	nodeAttrHCloudServerID = "unique.hostname"
)

// setupHCloudClient takes the passed config mapping and instantiates the
// required Hetzner Cloud client.
func (t *TargetPlugin) setupHCloudClient(config map[string]string) error {

	token, ok := config[configKeyToken]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyToken)
	}

	t.hcloud = hcloud.NewClient(hcloud.WithToken(token))
	return nil
}

// scaleOut adds HCloud servers up to desired count to match what the
// Autoscaler has deemed required.
func (t *TargetPlugin) scaleOut(ctx context.Context, servers []*hcloud.Server, count int64, config map[string]string) error {
	// Create a logger for this action to pre-populate useful information we
	// would like on all log lines.
	log := t.logger.With("action", "scale_out", "hcloud_group_id", config[configKeyGroupID],
		"desired_count", count)

	location, ok := config[configKeyLocation]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyLocation)
	}

	imageName, ok := config[configKeyImage]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyImage)
	}

	image, _, err := t.hcloud.Image.Get(ctx, imageName)
	if err != nil {
		return fmt.Errorf("couldn't retrieve HCloud image: %v", err)
	}

	if image == nil {
		return fmt.Errorf("couldn't retrieve HCloud image: %s", imageName)
	}

	userData, ok := config[configKeyUserData]
	if !ok {
		return fmt.Errorf("required config param %s not found", configKeyUserData)
	}

	b64UserDataEncoded := false
	if _, ok := config[configKeyB64UserDataEncoded]; ok {
		b64UserDataEncoded, err = strconv.ParseBool(config[configKeyB64UserDataEncoded])
		if err != nil {
			return fmt.Errorf("failed to parse %s parameter: %v", configKeyB64UserDataEncoded, err)
		}
	}

	if _, ok := config[configKeyServerType]; !ok {
		return fmt.Errorf("required config param %s not found", configKeyServerType)
	}

	if b64UserDataEncoded {
		userDataBytes, err := base64.StdEncoding.DecodeString(userData)
		userData = string(userDataBytes)
		if err != nil {
			return fmt.Errorf("failed to perform b64 decode of user data: %v", err)
		}
	}

	opts := hcloud.ServerCreateOpts{
		ServerType: &hcloud.ServerType{
			Name: config[configKeyServerType],
		},
		UserData: userData,
		Image:    image,
		Location: &hcloud.Location{
			Name: location,
		},
	}

	if datacenter, ok := config[configKeyDatacenter]; ok {
		opts.Datacenter = &hcloud.Datacenter{Name: datacenter}
	}

	if sshKeys, ok := config[configKeySSHKeys]; !ok {
		return fmt.Errorf("required config param %s not found", configKeySSHKeys)
	} else {
		for _, sshKeyValue := range strings.Split(sshKeys, ",") {
			sshKey, _, err := t.hcloud.SSHKey.Get(ctx, sshKeyValue)
			if err != nil {
				return fmt.Errorf("failed to get HCloud SSH key: %v", err)
			}
			if sshKey == nil {
				return fmt.Errorf("HCloud SSH key not found: %s", sshKeyValue)
			}
			opts.SSHKeys = append(opts.SSHKeys, sshKey)
		}
	}

	labels := make(map[string]string)

	if labelSelector, ok := config[configKeyLabels]; ok {
		labels, err = extractLabels(labelSelector)
		if err != nil {
			return fmt.Errorf("failed to parse labels: %v", err)
		}
	}

	labels[groupIDLabel] = config[configKeyGroupID]
	opts.Labels = labels

	if networks, ok := config[configKeyNetworks]; ok {
		for _, networkValue := range strings.Split(networks, ",") {
			network, _, err := t.hcloud.Network.Get(ctx, networkValue)
			if err != nil {
				return fmt.Errorf("failed to get HCloud Network: %v", err)
			}
			if network == nil {
				return fmt.Errorf("HCloud network not found: %s", networkValue)
			}
			opts.Networks = append(opts.Networks, network)
		}
	}

	f := func(ctx context.Context) (bool, error) {
		var results []hcloud.ServerCreateResult
		countDiff := count - int64(len(servers))
		var counter int64
		for counter < countDiff {
			id := uuid.New()
			suffix := strings.Replace(id.String(), "-", "", -1)[:defaultRandomSuffixLen]
			opts.Name = fmt.Sprintf("%s-%s", config[configKeyGroupID], suffix)
			result, _, err := t.hcloud.Server.Create(ctx, opts)
			if err != nil {
				log.Error("failed to create an HCloud server", err)
				break
			}
			results = append(results, result)
			counter++
		}
		var actionIDs []int
		for _, result := range results {
			if result.Action.Progress < 100 {
				actionIDs = append(actionIDs, result.Action.ID)
			}
		}
		_, _, err := t.ensureActionsComplete(ctx, actionIDs)
		if err != nil {
			log.Error("failed to wait till all HCloud create actions are ready", err)
		}
		labelSelector := fmt.Sprintf("%s=%s", groupIDLabel, config[configKeyGroupID])
		servers, err = t.getServers(ctx, labelSelector)
		if err != nil {
			return false, fmt.Errorf("failed to get a new servers count during instance scale out: %v", err)
		}
		serverCount := int64(len(servers))
		if serverCount == count {
			return true, nil
		}
		return false, fmt.Errorf("waiting for %v servers to create", count-serverCount)
	}

	return retry(ctx, defaultRetryInterval, defaultRetryLimit, f)
}

func (t *TargetPlugin) scaleIn(ctx context.Context, servers []*hcloud.Server, count int64, config map[string]string) (err error) {
	// Create a logger for this action to pre-populate useful information we
	// would like on all log lines.
	log := t.logger.With("action", "scale_in", "hcloud_group_id", config[configKeyGroupID])
	remoteIDs := []string{}
	for _, server := range servers {
		if server.Status == hcloud.ServerStatusRunning {
			remoteIDs = append(remoteIDs, server.Name)
		}
	}
	nodes, err := t.clusterUtils.RunPreScaleInTasksWithRemoteCheck(ctx, config, remoteIDs, int(count))
	if err != nil {
		return fmt.Errorf("failed to perform pre-scale Nomad scale in tasks: %v", err)
	}

	for _, node := range nodes {
		var serverInput hcloud.Server
		for _, server := range servers {
			if server.Name == node.RemoteResourceID {
				serverInput.ID = server.ID
				break
			}
		}
		_, err := t.hcloud.Server.Delete(ctx, &serverInput)
		if err != nil {
			log.Error("failed to delete a HCloud server",
				"server_id", node.RemoteResourceID, "node_id", node.NomadNodeID,
				"error", err)
		}
	}

	if err := t.clusterUtils.RunPostScaleInTasks(ctx, config, nodes); err != nil {
		return fmt.Errorf("failed to perform post-scale Nomad scale in tasks: %v", err)
	}

	return
}

func (t *TargetPlugin) getServers(ctx context.Context, labelSelector string) ([]*hcloud.Server, error) {
	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector,
			PerPage:       defaultPerPage,
		},
		Status: []hcloud.ServerStatus{hcloud.ServerStatusRunning},
	}
	servers, err := t.hcloud.Server.AllWithOpts(ctx, opts)
	if err != nil {
		t.logger.Error("error retrieving server", err)
		return nil, err
	}
	return servers, nil
}

func (t *TargetPlugin) ensureActionsComplete(ctx context.Context, ids []int) (successfulActions []int, failedActions []int, err error) {

	opts := hcloud.ActionListOpts{
		ID: ids,
	}

	f := func(ctx context.Context) (bool, error) {
		currentActions, _, err := t.hcloud.Action.List(ctx, opts)
		if err != nil {
			return false, err
		}

		// Reset the action IDs we are waiting to complete so we can
		// re-populate with a modified list later.
		var ids []int

		// Iterate each action, check the progress and add any incomplete
		// actions to the ID list for rechecking.
		for _, action := range currentActions {
			if action.Progress < 100 {
				ids = append(ids, action.ID)
			} else if action.Status == hcloud.ActionStatusError {
				failedActions = append(failedActions, action.ID)
				t.logger.Error("Hetzner cloud action id", action.ID, "failed with code", action.ErrorCode, "error", action.ErrorMessage)
			} else if action.Status == hcloud.ActionStatusSuccess {
				successfulActions = append(successfulActions, action.ID)
			}
		}

		// If we dont have any remaining IDs to check, we can finish.
		if len(ids) == 0 {
			return true, nil
		}
		return false, fmt.Errorf("waiting for %v actions to finish", len(ids))
	}

	err = retry(ctx, defaultRetryInterval, defaultRetryLimit, f)
	return
}

// hcloudNodeIDMap is used to identify the HCloud Server of a Nomad node using
// the relevant attribute value.
func hcloudNodeIDMap(n *api.Node) (string, error) {
	val, ok := n.Attributes[nodeAttrHCloudServerID]
	if !ok || val == "" {
		return "", fmt.Errorf("attribute %q not found", nodeAttrHCloudServerID)
	}
	return val, nil
}

func extractLabels(labelsStr string) (map[string]string, error) {
	labels := make(map[string]string)
	labelStrs := strings.Split(labelsStr, ",")
	for _, labelStr := range labelStrs {
		if labelStr == "" {
			continue
		}
		labelValues := strings.Split(labelStr, "=")
		if len(labelValues) == 2 {
			labels[strings.TrimSpace(labelValues[0])] = strings.TrimSpace(labelValues[1])
		} else {
			return nil, fmt.Errorf("failed to parse labels %s", labelsStr)
		}
	}
	return labels, nil
}
