---
layout: "docs"
page_title: "Secret Backend: azureservicebus"
sidebar_current: "docs-secrets-azureservicebus"
description: |-
  The Azure Service Bus secret backend for Vault generates database credentials to access Microsoft Sql Server.
---

# Azure Service Bus Secret Backend

Name: `azureservicebus`

The Azure Service Bus secret backend for Vault generates Service Bus Service Access 
Signature (SAS) Tokens dynamically based on configured roles, which corresponds to 
Shared Access Policies. 

You can learn more about using SAS Tokens to authenticate against Service Bus 
resources here: https://azure.microsoft.com/en-us/documentation/articles/service-bus-shared-access-signature-authentication/ 

SAS tokens expire automatically, so Vault cannot renew tokens.

This page will show a quick start for this backend. For detailed documentation
on every path, use `vault path-help` after mounting the backend.

## Quick Start

The first step to using the backend is to mount it.
Unlike the `generic` backend, the `azureservicebus` backend is not mounted by default.

```
$ vault mount azureservicebus
Successfully mounted 'azureservicebus' at 'azureservicebus'!
```

Next, we must configure Vault to know how which Service Bus resource we wish to connect to.
This is specified with resource name and namespace in `config/resource`:

```
$ vault write azureservicebus/config/resource \
  name=my_eventhub \
  namespace=my_service_bus_ns
Success! Data written to: azureservicebus/config/resource
```

We can configure the default expiry time for tokens generated
by Vault. This is done by writing `config/lease`:

```
$ vault write azureservicebus/config/lease \
    ttl=1h
Success! Data written to: azureservicebus/config/lease
```

Configure roles by specifying the Share Access Policy name and 
primary key in `roles/<role_name>`. You can get these from the 
classic Azure Portal.

```
$ vault write azureservicebus/roles/all \
    sas_policy_name=manage_send_listen_policy \
    sas_policy_key=your_policy_primary_key \
    ttl=15m
Success! Data written to: azureservicebus/roles/all
```

Expiry times can configured per role. If you have a Time-To-Live (`ttl`) time 
specified in the role configuration, tokens generated with that role will
respect the policy-specific policy. Otherwise, the default expiry time from 
config/lease is used. In the example, tokens read from the `all` policy will
always have a expiry time of 15 minutes instead of 1 hour. If ttl is not 
specified in role, the default expiry time is used.

SAS restricts each token to being valid until its declared time. Vault will 
not renew this token unlike some other secret backends, so clients need to 
request for a new one.

```
$ vault read azureservicebus/token/all
Key            	Value
lease_id       	azureservicebus/token/all/e94b071c-1fc8-6e8a-76df-b434bf9aa3e7
lease_duration 	900
lease_renewable	false
token          	SharedAccessSignature sr=https%3a%2f%2fmy_service_bus_ns.servicebus.windows.net%2fmy_eventhub&sig=NvCCJ&se=146534&skn=manage_send_listen_policy
```

By reading from the `token/all` path, Vault has generated a new
SAS token using the `all` role configuration, which will be valid for 15 minutes.

## API

### /azureservicebus/config/resource
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Configures the Service Bus resource name and namespace.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/config/resource`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">name</span>
        <span class="param-flags">required</span>
        Resource name.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">namespace</span>
        <span class="param-flags">required</span>
        Service Bus namespace.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azureservicebus/config/lease
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Configures the lease settings for generated token.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/config/lease`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">ttl</span>
        <span class="param-flags">required</span>
        The ttl value provided as a string duration
        with time suffix. Hour is the largest suffix.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azureservicebus/roles/
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Creates or updates the role definition.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">sas_policy_name</span>
        <span class="param-flags">required</span>
        The name of the Shared Access Policy this role is associated with.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">sas_policy_key</span>
        <span class="param-flags">required</span>
        The primary key of the Shared Access Policy.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">ttl</span>
        <span class="param-flags">optional</span>
        The role-specifc expiry time.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

#### GET

<dl class="api">
  <dt>Description</dt>
  <dd>
    Queries the role definition.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "data": {
        "sas_policy_name": "manage_send_listen_policy",
        "ttl":             0
      }
    }
    ```

  </dd>
</dl>

#### LIST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Returns a list of available roles. Only the role names are returned, not
    any values.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/roles/?list=true`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>

  ```javascript
  {
    "auth": null,
    "data": {
      "keys": ["all", "prod"]
    },
    "lease_duration": 2592000,
    "lease_id": "",
    "renewable": false
  }
  ```

  </dd>
</dl>

#### DELETE

<dl class="api">
  <dt>Description</dt>
  <dd>
    Deletes the role definition.
  </dd>

  <dt>Method</dt>
  <dd>DELETE</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azureservicebus/creds/
#### GET

<dl class="api">
  <dt>Description</dt>
  <dd>
    Generates a new set of dynamic credentials based on the named role.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/azureservicebus/creds/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "data": {
        "token": "SharedAccessSignature sr=https..."
      }
    }
    ```

  </dd>
</dl>
