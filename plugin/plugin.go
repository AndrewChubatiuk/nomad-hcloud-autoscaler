package plugin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	ent "github.com/go-playground/validator/v10/translations/en"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/plugins/target"
	"github.com/hashicorp/nomad-autoscaler/sdk"
	"github.com/hashicorp/nomad-autoscaler/sdk/helper/nomad"
	"github.com/hashicorp/nomad-autoscaler/sdk/helper/scaleutils"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

const (
	// pluginName is the unique name of the this plugin amongst Target plugins.
	pluginName = "hcloud-server"
)

var (
	PluginConfig = &plugins.InternalPluginConfig{
		Factory: func(l hclog.Logger) interface{} { return NewHCloudServerPlugin(l) },
	}

	pluginInfo = &base.PluginInfo{
		Name:       pluginName,
		PluginType: sdk.PluginTypeTarget,
	}

	validate = validator.New()
	eng      = en.New()
	uni      = ut.New(eng, eng)
)

// Assert that TargetPlugin meets the target.Target interface.
var _ target.Target = (*TargetPlugin)(nil)

// TargetPlugin is the Hetzner Cloud Server implementation of the target.Target interface.
type TargetPlugin struct {
	config HCloudPluginConfig
	logger hclog.Logger
	hcloud *hcloud.Client

	// clusterUtils provides general cluster scaling utilities for querying the
	// state of nodes pools and performing scaling tasks.
	clusterUtils *scaleutils.ClusterScaleUtils
}

// NewHCloudServerPlugin returns the Hetzner Cloud Server implementation of the target.Target
// interface.
func NewHCloudServerPlugin(log hclog.Logger) *TargetPlugin {
	return &TargetPlugin{
		logger: log,
	}
}

// SetConfig satisfies the SetConfig function on the base.Base interface.
func (t *TargetPlugin) SetConfig(config map[string]string) error {

	trans, _ := uni.GetTranslator("en")
	if err := ent.RegisterDefaultTranslations(validate, trans); err != nil {
		return err
	}
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("mapstructure"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	err := validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} value is not set in a {1} config", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		var configTypeName string
		fmt.Println(fe.Namespace())
		if strings.HasPrefix(fe.Namespace(), "HCloudTargetConfig") {
			configTypeName = "target"
		} else if strings.HasPrefix(fe.Namespace(), "HCloudPluginConfig") {
			configTypeName = "plugin"
		}
		t, _ := ut.T("required", fe.Field(), configTypeName)

		return t
	})

	if err != nil {
		return err
	}

	if err := Parse(config, &t.config); err != nil {
		return fmt.Errorf("failed to parse HCloud plugin config: %v", err)
	}

	t.setupHCloudClient()

	clusterUtils, err := scaleutils.NewClusterScaleUtils(nomad.ConfigFromNamespacedMap(config), t.logger)
	if err != nil {
		return err
	}

	// Store and set the remote ID callback function.
	t.clusterUtils = clusterUtils
	t.clusterUtils.ClusterNodeIDLookupFunc = t.hcloudNodeIDMap

	return nil
}

// PluginInfo satisfies the PluginInfo function on the base.Base interface.
func (t *TargetPlugin) PluginInfo() (*base.PluginInfo, error) {
	return pluginInfo, nil
}

// Scale satisfies the Scale function on the target.Target interface.
func (t *TargetPlugin) Scale(action sdk.ScalingAction, config map[string]string) error {

	// Hetzner Cloud can't support dry-run like Nomad, so just exit.
	if action.Count == sdk.StrategyActionMetaValueDryRunCount {
		return nil
	}

	ctx := context.Background()

	// Get Hetzner Cloud servers. This serves to both validate the config value is
	// correct and ensure the HCloud client is configured correctly. The response
	// can also be used when performing the scaling, meaning we only need to
	// call it once.
	var targetConfig HCloudTargetConfig
	if err := Parse(config, &targetConfig); err != nil {
		return fmt.Errorf("failed to parse HCloud target config: %v", err)
	}

	servers, err := t.getServers(ctx, &targetConfig)
	if err != nil {
		return fmt.Errorf("failed to get HCloud servers: %v", err)
	}

	// The Hetzner Cloud servers require different details depending on which
	// direction we want to scale. Therefore calculate the direction and the
	// relevant number so we can correctly perform the HCloud work.
	num, direction := t.calculateDirection(int64(len(servers)), action.Count)

	switch direction {
	case "in":
		err = t.scaleIn(ctx, servers, num, config, &targetConfig)
	case "out":
		err = t.scaleOut(ctx, servers, num, config, &targetConfig)
	default:
		t.logger.Info("scaling not required", "hcloud_name_prefix", targetConfig.GroupID,
			"current_count", len(servers), "strategy_count", action.Count)
		return nil
	}

	// If we received an error while scaling, format this with an outer message
	// so its nice for the operators and then return any error to the caller.
	if err != nil {
		err = fmt.Errorf("failed to perform scaling action: %v", err)
	}
	return err
}

// Status satisfies the Status function on the target.Target interface.
func (t *TargetPlugin) Status(config map[string]string) (*sdk.TargetStatus, error) {

	ready, err := t.clusterUtils.IsPoolReady(config)
	if err != nil {
		return nil, fmt.Errorf("failed to run Nomad node readiness check: %v", err)
	}
	if !ready {
		return &sdk.TargetStatus{Ready: ready}, nil
	}

	var targetConfig HCloudTargetConfig
	if err := Parse(config, &targetConfig); err != nil {
		return nil, fmt.Errorf("failed to parse HCloud target config: %v", err)
	}

	ctx := context.Background()

	servers, err := t.getServers(ctx, &targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of hetzner servers: %v", err)
	}

	serverCount := int64(len(servers))

	// Set our initial status. The asg.Status field is only set when the ASG is
	// being deleted
	resp := sdk.TargetStatus{
		Ready: true,
		Count: serverCount,
		Meta:  make(map[string]string),
	}

	return &resp, nil
}

func (t *TargetPlugin) calculateDirection(current, strategyDesired int64) (int64, string) {

	if strategyDesired < current {
		return current - strategyDesired, "in"
	}
	if strategyDesired > current {
		return strategyDesired, "out"
	}
	return 0, ""
}
