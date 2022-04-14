#!/bin/bash
set -e

systemctl restart consul

export CONSUL_HTTP_ADDR=http://$(ip -o -4 addr list ${interface} | head -n1 | awk '{print $4}' | cut -d/ -f1):8500
until [[ $(curl -s -f $CONSUL_HTTP_ADDR/v1/status/leader) =~ \"[0-9.]+:8300\" ]] ; do
  echo "Waiting for consul to become available..."
  sleep 5
done
%{ if length(servers) == 0 }
export CONSUL_HTTP_TOKEN=$(consul acl bootstrap -format=json | jq -r '.SecretID')
%{ else }
export CONSUL_HTTP_TOKEN=${consul_token}
%{ endif }
echo "CONSUL_HTTP_TOKEN=$CONSUL_HTTP_TOKEN" >> /etc/nomad.d/nomad.env
echo "CONSUL_HTTP_ADDR=$CONSUL_HTTP_ADDR" >> /etc/nomad.d/nomad.env
echo "CONSUL_HTTP_TOKEN=$CONSUL_HTTP_TOKEN" >> /etc/consul.d/consul.env

systemctl restart nomad

export NOMAD_ADDR=http://$(ip -o -4 addr list ${interface} | head -n1 | awk '{print $4}' | cut -d/ -f1):4646
until [[ $(curl -s -f $NOMAD_ADDR/v1/status/leader) =~ \"[0-9.]+:4647\" ]] ; do
  echo "Waiting for nomad to become available..."
  sleep 5
done
%{ if length(servers) == 0 }
export NOMAD_TOKEN=$(nomad acl bootstrap -json | jq -r '.SecretID')
%{ else }
export NOMAD_TOKEN=${ nomad_token }
%{ endif }
echo "NOMAD_TOKEN=$NOMAD_TOKEN" >> /etc/nomad.d/nomad.env
echo "NOMAD_TOKEN=$NOMAD_TOKEN" >> /etc/consul.d/consul.env

systemctl enable consul nomad

cat > /tmp/creds.json <<EOL
{
    "nomad": "$${NOMAD_TOKEN}",
    "consul": "$${CONSUL_HTTP_TOKEN}"
}
EOL