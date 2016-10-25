#!/bin/bash

vault=$1
ips=$2

OFS=$IFS
IFS=",";for ip in $ips; do
    token=$(azure keyvault secret show $vault vault-key-1 --json | jq .value | tr -d '"' | sed 's/\\n//g' | base64 -D)
    vault unseal -address=https://${ip} -tls-skip-verify $token
    token=$(azure keyvault secret show $vault vault-key-2 --json | jq .value | tr -d '"' | sed 's/\\n//g' | base64 -D)
    vault unseal -address=https://${ip} -tls-skip-verify $token
    token=$(azure keyvault secret show $vault vault-key-3 --json | jq .value | tr -d '"' | sed 's/\\n//g' | base64 -D)
    vault unseal -address=https://${ip} -tls-skip-verify $token
done
IFS=$OFS
