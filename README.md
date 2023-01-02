# Hetzner Cloud Server Target Plugin

The `hcloud-server` target plugin allows for the scaling of the Nomad cluster clients via manipulating [Hetzner Cloud Servers][hcloud_servers].

## Agent Configuration Options

To use the `hcloud-server` target plugin, the agent configuration needs to be populated with the appropriate target block.

```hcl
target "hcloud-server" {
  driver = "hcloud-server"
  config = {
    hcloud_token = "YOUR_HCLOUD_TOKEN"
  }
}
```

- `hcloud_token` `(string: required)` - The [Hetzner Cloud token][hcloud_token] used to authenticate to connect to and where resources should be managed.

- `hcloud_random_suffix_len` `(string: "10")` - Random Server name suffix length

- `hcloud_retry_interval` `(string: "1m")` - Hetzner Cloud API retry interval

- `hcloud_retry_limit` `(string: "5")` - Hetzner Cloud API retry limit

- `hcloud_items_per_page` `(string: "50")` - Hetzner Cloud API request page size

- `hcloud_group_id_label_selector` `(string: "group-id")` - Server group id label selector

- `hcloud_node_attr_id` `(string: "unique.hostname")` - Nomad Node attribute id

### Nomad ACL

When using a Nomad cluster with ACLs enabled, the plugin will require an ACL token which provides the following permissions:

```hcl
node {
  policy = "write"
}
```

## Policy Configuration Options

```hcl
check "hashistack-allocated-cpu" {
  # ...
  target "hcloud-server" {
    datacenter                   = "XXX"
    node_class                   = "XXX"
    node_drain_deadline          = "5m"
    node_purge                   = "true"
    node_selector_strategy       = "newest_create_index"
    hcloud_location              = "XXX"
    hcloud_image                 = "XXX"
    hcloud_user_data             = "#cloud-config\npackages:\n - jq"
    hcloud_b64_user_data_encoded = "false"
    hcloud_ssh_keys              = "XXX"
    hcloud_server_type           = "cx11"
    hcloud_group_id              = "XXX"
    hcloud_labels                = "XXX_node=true"
    hcloud_networks              = "XXX"
  }
  # ...
}
```

- `hcloud_location` `(string: "")` - ID or name of [Location][hcloud_location] to create Server in (must not be used together with `hcloud_datacenter`).

- `hcloud_datacenter` `(string: "")` - ID or name of [Datacenter][hcloud_datacenter] to create Server in (must not be used together with `hcloud_location`).

- `hcloud_firewalls` `(string: "")` - Comma-separated list of [Firewall][hcloud_firewall] IDs

- `hcloud_placement_group` `(string: "")` - [Placement Group][hcloud_placement_group] ID

- `hcloud_image` `(string: required)` - ID or name of the [Image][hcloud_image] the Server is created from.

- `hcloud_group_id` `(string: required)` - Server group name used for filtering targeted HCloud hosts. `group-id` label is attached to a server during creation.

- `hcloud_user_data` `(string: required)` - [Cloud-Init][cloud_init] user data to use during Server creation. This field is limited to 32KiB.

- `hcloud_b64_user_data_encoded` `(string: "false")` - Identifies if `hcloud_user_data` is base64 encoded or not.

- `hcloud_ssh_keys` `(string: required)` - Comma-separated IDs or names of SSH keys which should be injected into the server at creation time.

- `hcloud_labels` `(string: "")` - User-defined labels (key-value pairs) string in a format `key1=value1,key2=value2,...,keyN=valueN`.

- `hcloud_networks` `(string: "")` - [Network][hcloud_networks] IDs which should be attached to the server private network interface at the creation time.

- `datacenter` `(string: "")` - The Nomad client [datacenter][nomad_datacenter] identifier used to group nodes into a pool of resource.

- `node_class` `(string: "")` - The Nomad [client node class][nomad_node_class] identifier used to group nodes into a pool of resource.

- `node_drain_deadline` `(duration: "15m")` The Nomad [drain deadline][nomad_node_drain_deadline] to use when performing node draining actions.

- `node_drain_ignore_system_jobs` `(bool: "false")` A boolean flag used to control if system jobs should be stopped when performing node draining actions.

- `node_purge` `(bool: "false")` A boolean flag to determine whether Nomad clients should be [purged][nomad_node_purge] when performing scale in actions.

- `node_selector_strategy` `(string: "least_busy")` The strategy to use when selecting nodes for termination. Refer to the [node selector strategy][node_selector_strategy] documentation for more information.

[hcloud_servers]: https://docs.hetzner.com/cloud/servers
[hcloud_datacenter]: https://www.hetzner.com/unternehmen/rechenzentrum
[hcloud_token]: https://docs.hetzner.com/dns-console/dns/general/api-access-token/
[hcloud_location]: https://docs.hetzner.com/cloud/general/locations/
[hcloud_placement_group]: https://docs.hetzner.com/cloud/placement-groups/overview/
[hcloud_image]: https://docs.hetzner.com/robot/dedicated-server/operating-systems/standard-images/
[hcloud_networks]: https://docs.hetzner.com/cloud/networks/overview
[hcloud_firewall]: https://docs.hetzner.com/robot/dedicated-server/firewall/
[cloud_init]: https://cloudinit.readthedocs.io/en/latest/
[nomad_datacenter]: /docs/configuration#datacenter
[nomad_node_class]: /docs/configuration/client#node_class
[nomad_node_drain_deadline]: /api-docs/nodes#deadline
[nomad_node_purge]: /api-docs/nodes#purge-node
[node_selector_strategy]: /tools/autoscaling/internals/node-selector-strategy

## Demo
Run `terraform apply` in [demo](demo/setup) folder to create: 
 - nomad server which runs services for:
    - nomad-autoscaler
    - prometheus
    - redis

Autoscaler scales hcloud nodes for redis. After successful run both Nomad and Consul are wide-world open and credentials for both you can find in terraform output
