package plugin

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"github.com/google/uuid"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/mitchellh/mapstructure"
)

func decodeFunc(
	f reflect.Type,
	t reflect.Type,
	data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t == reflect.TypeOf(time.Duration(5)) {
		return time.ParseDuration(data.(string))
	}
	var result reflect.Value
	switch t.Kind() {
	case reflect.Ptr:
		value, err := decodeFunc(f, t.Elem(), data)
		if err != nil {
			return nil, err
		}
		result = reflect.New(t.Elem())
		result.Elem().Set(reflect.ValueOf(value))
	case reflect.Struct:
		result = reflect.Indirect(reflect.New(t))
		idValue := result.FieldByName("ID")
		nameValue := result.FieldByName("Name")
		if !(idValue.IsValid() && nameValue.IsValid()) {
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				value, err := decodeFunc(f, field.Type, data)
				if err != nil {
					return nil, err
				}
				if reflect.TypeOf(value) == field.Type {
					result.Field(i).Set(reflect.ValueOf(value))
				}
			}
		} else {
			if id, err := strconv.Atoi(data.(string)); err == nil {
				idValue.Set(reflect.ValueOf(id))
			} else {
				nameValue.Set(reflect.ValueOf(data.(string)))
			}
		}
	case reflect.Slice:
		sliceType := reflect.SliceOf(t.Elem())
		result = reflect.MakeSlice(sliceType, 0, 0)
		for _, sliceStr := range strings.Split(data.(string), ",") {
			if strings.TrimSpace(sliceStr) != "" {
				value, err := decodeFunc(f, t.Elem(), sliceStr)
				if err != nil {
					return nil, err
				}
				result = reflect.Append(result, reflect.ValueOf(value))
			}
		}

	case reflect.Map:
		mapType := reflect.MapOf(t.Key(), t.Elem())
		result = reflect.MakeMap(mapType)
		mapStrs := strings.Split(data.(string), ",")
		for _, kvStr := range mapStrs {
			kvPair := strings.Split(kvStr, "=")
			key := strings.TrimSpace(kvPair[0])
			value := strings.TrimSpace(kvPair[1])
			if len(kvPair) == 2 && key != "" && value != "" {
				result.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			} else {
				return nil, fmt.Errorf("failed to parse map %v", data)
			}
		}
	}
	if result.IsValid() {
		return result.Interface(), nil
	}
	return data, nil
}

type HCloudPluginConfig struct {
	Token                string        `mapstructure:"hcloud_token" validate:"required"`
	RandomSuffixLen      int           `mapstructure:"hcloud_random_suffix_len" default:"10"`
	RetryInterval        time.Duration `mapstructure:"hcloud_retry_interval" default:"60s"`
	RetryLimit           int           `mapstructure:"hcloud_retry_limit" default:"5"`
	ItemsPerPage         int           `mapstructure:"hcloud_items_per_page" default:"50"`
	GroupIDLabelSelector string        `mapstructure:"hcloud_group_id_label_selector" default:"group-id"`
	NodeAttrID           string        `mapstructure:"hcloud_node_attr_id" default:"unique.hostname"`
}

type HCloudTargetConfig struct {
	Datacenter         *hcloud.Datacenter             `mapstructure:"hcloud_datacenter" validate:"required_without=Location"`
	Location           *hcloud.Location               `mapstructure:"hcloud_location" validate:"required_without=Datacenter"`
	PlacementGroup     *hcloud.PlacementGroup         `mapstructure:"hcloud_placement_group"`
	Firewalls          []*hcloud.ServerCreateFirewall `mapstructure:"hcloud_firewalls"`
	Image              *hcloud.Image                  `mapstructure:"hcloud_image" default:"{\"Name\": \"ubuntu-20.04\"}" validate:"required"`
	UserData           string                         `mapstructure:"hcloud_user_data" validate:"required"`
	SSHKeys            []*hcloud.SSHKey               `mapstructure:"hcloud_ssh_keys" validate:"required"`
	Labels             map[string]string              `mapstructure:"hcloud_labels"`
	ServerType         *hcloud.ServerType             `mapstructure:"hcloud_server_type" default:"{\"Name\":\"cx11\"}" validate:"required"`
	GroupID            string                         `mapstructure:"hcloud_group_id" validate:"required"`
	Networks           []*hcloud.Network              `mapstructure:"hcloud_networks"`
	B64UserDataEncoded bool                           `mapstructure:"hcloud_b64_user_data_encoded"`
}

func (tc *HCloudTargetConfig) GetSelector(labelName string) string {
	selectorSlice := []string{
		fmt.Sprintf("%s=%s", labelName, tc.GroupID),
	}

	for key, value := range tc.Labels {
		selectorSlice = append(selectorSlice, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(selectorSlice, ",")
}

func (tc *HCloudTargetConfig) RandomName(suffixLen int) string {
	id := uuid.New()
	suffix := strings.Replace(id.String(), "-", "", -1)[:suffixLen]
	return fmt.Sprintf("%s-%s", tc.GroupID, suffix)
}

func Parse(input interface{}, output interface{}) error {

	if err := defaults.Set(output); err != nil {
		return err
	}

	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		DecodeHook:       CustomDecodeHookFunc(),
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	err = decoder.Decode(input)
	if err != nil {
		return err
	}

	err = validate.Struct(output)
	if err != nil {
		return err
	}
	return nil
}

func CustomDecodeHookFunc() mapstructure.DecodeHookFunc {
	return decodeFunc
}
