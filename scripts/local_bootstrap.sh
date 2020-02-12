#!/bin/bash

function vault_unseal() {
    sleep 10
    INIT_DATA=$(vault operator init --key-shares=1 --key-threshold=1 --format json)
    UNSEAL_KEY=$(echo "${INIT_DATA}" | jq -r '.unseal_keys_b64[0]')
    ROOT_TOKEN=$(echo "${INIT_DATA}" | jq -r '.root_token')
    vault operator unseal ${UNSEAL_KEY} > /dev/null
    echo ${ROOT_TOKEN}
}

function origin_token() {
    echo '
# TODO: remove create update delete when vsync code is not checking for all capabilities of a token
path "secret/*" {
  capabilities = ["create","update","read","list","delete"]
}
path "multipaas/*" {
  capabilities = ["create","update","read","list","delete"]
}
path "sys/mounts" {
    capabilities = ["read","list"]
}
' > /tmp/vsync_origin
    vault policy write vsync_origin /tmp/vsync_origin
    vault token create --policy vsync_origin --ttl 2h
    echo "Copy the token and place in config file"
}

function destination_token() {
    echo '
# TODO: remove create update delete when vsync code is not checking for all capabilities of a token
path "secret/*" {
  capabilities = ["create","update","read","list","delete"]
}
path "multipaas/*" {
  capabilities = ["create","update","read","list","delete"]
}
path "sys/mounts" {
    capabilities = ["read","list"]
}
' > /tmp/vsync_destination
    vault policy write vsync_destination /tmp/vsync_destination
    vault token create --policy vsync_destination --ttl 2h
    echo "Copy the token and place in config file"
}

# destroy
docker stop originC && docker rm originC
docker stop agent1 && docker rm agent1
docker stop originV && docker rm originV

docker stop destinationC && docker rm destinationC
docker stop agent2 && docker rm agent2
docker stop destinationV && docker rm destinationV
docker network rm vsync

# create
docker network create vsync

# origin
docker run -d --name originC --network vsync -p 6500:8500 -p 6600:8600 consul agent --node originC --server --ui --bootstrap --client 0.0.0.0 --datacenter dc1
docker run -d --name agent1 --network vsync consul agent --node agent1 --retry-join originC
docker run -d --name originV --cap-add IPC_LOCK --volume "${PWD}"/scripts/originV_config.json:/tmp/originV_config.json --network vsync -p 6200:8200 vault server --config /tmp/originV_config.json
export VAULT_ADDR=http://localhost:6200
origin_ROOT_TOKEN=$(vault_unseal)
echo "root token for http://localhost:6200 : ${origin_ROOT_TOKEN}"
vault login ${origin_ROOT_TOKEN} > /dev/null
vault audit enable file file_path=/vault/logs/vault_audit.log
vault secrets enable -path=multipaas --version 2 kv
vault secrets enable -path=secret --version 2 kv
origin_token

# destination
docker run -d --name destinationC --network vsync -p 7500:8500 -p 7600:8600 consul agent --node destinationC --server --ui --bootstrap --retry-join-wan originC --client 0.0.0.0 --datacenter dc2
docker run -d --name agent2 --network vsync consul agent --node agent2 --retry-join destinationC
docker run -d --name destinationV --cap-add IPC_LOCK --volume "${PWD}"/scripts/destinationV_config.json:/tmp/destinationV_config.json --network vsync -p 7200:8200 vault server --config /tmp/destinationV_config.json
export VAULT_ADDR=http://localhost:7200
destination_ROOT_TOKEN=$(vault_unseal)
echo "root token for http://localhost:7200 : ${destination_ROOT_TOKEN}"
vault login ${destination_ROOT_TOKEN} > /dev/null
vault audit enable file file_path=/vault/logs/vault_audit.log
vault secrets enable -path=multipaas --version 2 kv
vault secrets enable -path=secret --version 2 kv
destination_token

# populate data
# install: brew install parallel
# update seq 2 -> seq 10000 for load testing
seq 2 | parallel --eta -j+0 "curl -Ssl -H 'X-Vault-Token: ${origin_ROOT_TOKEN}' -H 'Content-Type: application/json' http://127.0.0.1:6200/v1/secret/data/multipaas/test/{} -X POST -d '{\"data\":{\"chumma\":\"bar\"}}' > /dev/null"