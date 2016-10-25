#!/bin/sh

#TLS CA certificate and TLS cert file need to be concat'ed together https://www.vaultproject.io/docs/config/#tls_cert_file
echo "Concat TLS primary certificate and CA cert"
cat $TLS_CERT $TLS_CA > /etc/certs/tls_ca_cert.crt

sed -e "s|<<TLS_KEY>>|$TLS_KEY|g" /etc/vault.conf.tmp > /etc/vault.conf

if [[ $BACKEND == "azure" && -z $AZURE_ACCOUNT_KEY ]]; then
  echo "Reading Azure creds from file"
  AZURE_ACCOUNT_NAME=$(cat /etc/azurecreds | awk -F ":" '/name/ {print $2}')
  AZURE_ACCOUNT_KEY=$(cat /etc/azurecreds | awk -F ":" '/key/ {print $2}')
fi

if [[ "$TLS_DISABLE" == "true" ]]; then
  BACKEND_CONFIG = ""
  sed -i "s/<<TLS_DISABLE>>/tls_disable = 1/g" /etc/vault.conf
  export VAULT_ADDR=http://vault.yammer.core:8200
else
  sed -i "s/<<TLS_DISABLE>>//g" /etc/vault.conf
  export VAULT_ADDR=https://vault.yammer.core:8200
fi

BACKEND=${BACKEND:-zookeeper}

if [[ "$BACKEND" == "zookeeper" ]]; then
  BACKEND_CONFIG="path = \"/vault\"\n  address = \"$ZK_HOSTS\"\n  advertise_addr = \"https://$HOST\"\n"
  HA_BACKEND=""

elif [[ "$BACKEND" == "azure" ]]; then
  BACKEND_CONFIG="container = \"vault\"\n  accountName = \"$AZURE_ACCOUNT_NAME\"\n  accountKey = \"$AZURE_ACCOUNT_KEY\"\n"
  HA_BACKEND="ha_backend \"zookeeper\" {\n  path = \"/vault\"\n  address = \"$ZK_HOSTS\"\n  advertise_addr = \"https://$HOST\"\n}"
fi
  #statements

sed -i "s|<<BACKEND>>|$BACKEND|g" /etc/vault.conf
sed -i "s|<<HA_BACKEND>>|$HA_BACKEND|g" /etc/vault.conf
sed -i "s|<<BACKEND_CONFIG>>|$BACKEND_CONFIG|g" /etc/vault.conf

echo "127.0.0.1    vault.yammer.core" >> /etc/hosts

cat /etc/certs/cachain.crt >> /etc/ssl/certs/ca-certificates.crt

if [[ "$DEV" == "true" ]]; then
  mkdir -p /vault-dev
  if [[ "$BACKEND" == "zookeeper" ]]; then
    vault server -config /etc/vault.conf > /vault.log 2>&1 &
  else
    vault server -config /etc/dev.conf > /vault.log 2>&1 &
  fi

  echo "Waiting for Vault to become ready"
  sleep 5
  while [[ "$(curl -s https://127.0.0.1:8200/v1/sys/seal-status -k  | grep -c "could not connect to a server")" != "0" ]]; do
    echo "."
    sleep 2
  done

  export VAULT_ADDR=https://127.0.0.1:8200
  echo "Started server in dev mode. Version: $(vault version)"
  vault init -tls-skip-verify > /vault-dev/init

  if [[ "$?" != "0" ]]; then
    TOKEN="dev-root-token"
  else
    TOKEN=$(cat /vault-dev/init | awk '/Root Token/ {print $4}')
  fi

  echo "Found root token: $TOKEN"

  vault unseal -tls-skip-verify $(cat /vault-dev/init | awk '/Key 1/ {print $4}')
  vault unseal -tls-skip-verify $(cat /vault-dev/init | awk '/Key 2/ {print $4}')
  vault unseal -tls-skip-verify $(cat /vault-dev/init | awk '/Key 3/ {print $4}')

  echo "Waiting for token update"
  while [[ "$success" != "1" ]]; do
    success=$(curl -XPOST https://127.0.0.1:8200/v1/auth/token/create -sk -d '{"id": "dev-root-token", "policies":["root"]}' -H "X-Vault-Token: $TOKEN" | grep -c "client_token")
    echo "."
    sleep 2
  done

  vault auth -tls-skip-verify dev-root-token

  tail -f /vault.log
else
  echo "starting metric collector"
  python /vault_check.py --hostname $HOST --chipper $CHIPPER_URL &
  vault server -config /etc/vault.conf &
  ca-serve
fi
