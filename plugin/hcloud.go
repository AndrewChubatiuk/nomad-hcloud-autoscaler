package plugin

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

// setupHCloudClient takes the passed config mapping and instantiates the
// required Hetzner Cloud client.
func (t *TargetPlugin) setupHCloudClient() {
	t.hcloud = hcloud.NewClient(hcloud.WithToken(t.config.Token))
}

// scaleOut adds HCloud servers up to desired count to match what the
// Autoscaler has deemed required.
func (t *TargetPlugin) scaleOut(ctx context.Context, servers []*hcloud.Server, count int64, config map[string]string, targetConfig *HCloudTargetConfig) error {
	// Create a logger for this action to pre-populate useful information we
	// would like on all log lines.
	log := t.logger.With("action", "scale_out", "hcloud_group_id", targetConfig.GroupID,
		"desired_count", count)

	userData := targetConfig.UserData

	if targetConfig.B64UserDataEncoded {
		userDataBytes, err := base64.StdEncoding.DecodeString(userData)
		userData = string(userDataBytes)
		if err != nil {
			return fmt.Errorf("failed to perform b64 decode of user data: %v", err)
		}
	}

	opts := hcloud.ServerCreateOpts{
		UserData:       userData,
		Image:          targetConfig.Image,
		Datacenter:     targetConfig.Datacenter,
		Location:       targetConfig.Location,
		ServerType:     targetConfig.ServerType,
		PlacementGroup: targetConfig.PlacementGroup,
		Firewalls:      targetConfig.Firewalls,
		SSHKeys:        targetConfig.SSHKeys,
		Labels:         targetConfig.Labels,
		Networks:       targetConfig.Networks,
	}

	opts.Labels[t.config.GroupIDLabelSelector] = targetConfig.GroupID

	f := func(ctx context.Context) (bool, error) {
		var results []hcloud.ServerCreateResult
		countDiff := count - int64(len(servers))
		var counter int64
		for counter < countDiff {
			opts.Name = targetConfig.RandomName(t.config.RandomSuffixLen)
			log.Info("Creating server with name", opts.Name)
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
		servers, err = t.getServers(ctx, targetConfig)
		if err != nil {
			return false, fmt.Errorf("failed to get a new servers count during instance scale out: %v", err)
		}
		serverCount := int64(len(servers))
		if serverCount == count {
			return true, nil
		}
		return false, fmt.Errorf("waiting for %v servers to create", count-serverCount)
	}

	return retry(ctx, t.config.RetryInterval, t.config.RetryLimit, f)
}

func (t *TargetPlugin) scaleIn(ctx context.Context, servers []*hcloud.Server, count int64, config map[string]string, targetConfig *HCloudTargetConfig) (err error) {
	// Create a logger for this action to pre-populate useful information we
	// would like on all log lines.
	log := t.logger.With("action", "scale_in", "hcloud_group_id", targetConfig.GroupID)
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

func (t *TargetPlugin) getServers(ctx context.Context, targetConfig *HCloudTargetConfig) ([]*hcloud.Server, error) {
	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: targetConfig.GetSelector(t.config.GroupIDLabelSelector),
			PerPage:       t.config.ItemsPerPage,
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

	err = retry(ctx, t.config.RetryInterval, t.config.RetryLimit, f)
	return
}

// hcloudNodeIDMap is used to identify the HCloud Server of a Nomad node using
// the relevant attribute value.
func (t *TargetPlugin) hcloudNodeIDMap(n *api.Node) (string, error) {
	val, ok := n.Attributes[t.config.NodeAttrID]
	if !ok || val == "" {
		return "", fmt.Errorf("attribute %q not found", t.config.NodeAttrID)
	}
	return val, nil
}
