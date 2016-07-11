---
layout: "docs"
page_title: "Secret Backend: azuresql"
sidebar_current: "docs-secrets-azuresql"
description: |-
  The Azure SQL secret backend for Vault generates database credentials to access Azure SQL Server.
---

# Azure SQL Secret Backend

Name: `azuresql`

The Azure SQL secret backend for Vault generates database credentials
dynamically based on configured roles. This means that services that need
to access a database no longer need to hardcode credentials: they can request
them from Vault, and use Vault's leasing mechanism to more easily roll keys.
And with every service accessing the database using unique credentials, it 
makes auditing much easier when questionable data access is discovered: you 
can track it down to the specific instance of a service based on the SQL username.

Vault makes use of its own internal revocation system to ensure that users
become invalid within a reasonable time of the lease expiring.

This works almost exactly like the MSSQL Secret Backend, except that it has an
additional feature: When services request for a role, they have the option to give their
host IP address, so that a firewall rule for the role can be created and managed by
vault. It only supports Azure SQL Database V12, as it assumes all users are created 
in contained database user model.

This page will show a quick start for this backend. For detailed documentation
on every path, use `vault path-help` after mounting the backend.

## Quick Start

The first step to using the azuresql backend is to mount it.
Unlike the `generic` backend, the `azuresql ` backend is not mounted by default.

```
$ vault mount azuresql
Successfully mounted 'azuresql' at 'azuresql'!
```

Next, we must configure Vault to know how to connect to an Azure SQL instance. 
This is done by providing a DSN (Data Source Name):

```
$ vault write azuresql/config/connection \
connection_string="server=azure_sql_server.database.windows.net;port=1433;Database=my_azure_db;user id=server_admin_login@azure_sql_server;password=Password!;Application Name=MyAppName"
Success! Data written to: azuresql/config/connection
```

In this case, we've configured Vault with the user "server_admin_login" and password "Password!",
connecting to an instance at "localhost" on port 1433. The server admin user was created
during resource launch from the Azure portal, but you can use any user with privileges to create
users and grant permissions for the particular database you want to to connect to.

If firewall rules need to be created, then we need to configure Vault to access Azure management 
settings. This is needed to manage firewall rules for each generated credentials.

```
$ vault write azuresql/config/subscription \
 subscription_id=xxxx000-xxxx-xxxx-xxxx-xxxx1111xxxx \
 server=azure_sql_server \
 management_cert=$(cat azure_mgmt_cert.pem | base64)
Success! Data written to: azuresql/config/subscription
```

Here, we've specified the name of the Azure SQL server, the subscription ID of the 
server, and the absolute path of the management PEM certificate of the subscription.
This allows Vault to create and delete firewall rules on the Azure SQL server.
The management certificate file can be obtained and extracted in this way:
http://stuartpreston.net/2015/02/retrieving-microsoft-azure-management-certificates-for-use-in-cross-platform-automationprovisioning-tools/


Optionally, we can configure the lease settings for credentials generated
by Vault. This is done by writing to the `config/lease` key:

```
$ vault write azuresql/config/lease \
    ttl=1h \
    ttl_max=24h
Success! Data written to: azuresql/config/lease
```

This restricts each credential to being valid or leased for 1 hour
at a time, with a maximum use period of 24 hours. This forces an
application to renew their credentials at least hourly, and to recycle
them once per day.

The next step is to configure a role. A role is a logical name that maps
to a policy used to generate those credentials. For example, lets create
a "readonly" role:

```
$ vault write azuresql/roles/readonly \
    sql="CREATE USER [{{name}}] WITH PASSWORD = '{{password}}'; GRANT SELECT ON SCHEMA::vault TO [{{name}}]"
Success! Data written to: azuresql/roles/readonly
```

By writing to the `roles/readonly` path we are defining the `readonly` role.
This role will be created by evaluating the given `sql` statements. By
default, the `{{name}}` and `{{password}}` fields will be populated by
Vault with dynamically generated values. 

This backend assumes that all users are to be created under the contained 
database model, which uses a similar syntax like the example shown. 
`GRANT` queries can be used to customize the privileges of the role.

To generate a new set of credentials without adding firewall rules, we just 
read from that role:
```
vault read azuresql/creds/readonly
Key            	Value
lease_id       	azuresql/creds/readonly/a80c8cff-7d4f-edcf-37b8-c341506a966c
lease_duration 	60
lease_renewable	true
fwrule
jdbc           	jdbc:sqlserver://azure_sql_server.database.windows.net:1433;database=my_azure_db;user=root-cabd2194-45af-5d9c-84ee-6c9adb448286@azure_sql_server.database.windows.net;password=121534cf-3118-8e33-ce32-9fa0dda197bb;encrypt=true;trustServerCertificate=false;hostNameInCertificate=*.database.windows.net;loginTimeout=30;
password       	121534cf-3118-8e33-ce32-9fa0dda197bb
username       	root-cabd2194-45af-5d9c-84ee-6c9adb448286
```

By reading from the `creds/readonly` path, Vault has generated a new
set of credentials using the `readonly` role configuration. Here we
see the dynamically generated username and password, along with a one
hour lease. A jdbc string is generated with the credentials for convenience.

If Azure SQL Server is set up with a Firewall that does not allow all 
incoming password, then in order for a user to be able to use the generated 
credentials, we need to add a firewall rule. The IP to be added to the 
rule is appended to the `creds/readonly` path:

```
$ vault read azuresql/creds/readonly/112.22.25.32
Key            	Value
lease_id       	azuresql/creds/readonly/112.22.25.32/c30b8fa5-6c1c-4c38-53bd-c597133645e0
lease_duration 	60
lease_renewable	true
fwrule         	0b9ebcb5-801c-d85c-9aa1-e02e25558b91-112.22.25.32
password       	38c82607-1e1f-0321-603d-d295a53b18dd
username       	root-0b9ebcb5-801c-d85c-9aa1-e02e25558b91
```
This requires the `config/subscription` path to be set up with Azure SQL Server 
management rights.

## API

### /azuresql/config/connection
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Configures the connection DSN used to communicate with Azure SQL Database.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azuresql/config/connection`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">connection_string</span>
        <span class="param-flags">required</span>
        The DSN to connect to the Azure SQL Database.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">max_open_connections</span>
        <span class="param-flags">optional</span>
        Maximum number of open connections to the database.
	Defaults to 2.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">verify_connection</span>
        <span class="param-flags">optional</span>
	If set, connection_string is verified by actually connecting to the database.
	Defaults to true.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azuresql/config/subscription
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Configures the Azure Subscription information of the Azure SQL server.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azuresql/config/subscription`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">subscription_id</span>
        <span class="param-flags">required</span>
	Azure Subscription ID.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">management_cert</span>
        <span class="param-flags">required</span>
	Management certificate of the subscription, as a base64 encoded PEM file.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">server</span>
        <span class="param-flags">required</span>
	Name of the Azure SQL Server.
      </li>
    </ul>
  </dd>
  <dd>
    <ul>
      <li>
        <span class="param">verify</span>
        <span class="param-flags">optional</span>
	If set, the subscription details is verified by connecting to Azure.
	Defaults to true.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>


### /azuresql/config/lease
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Configures the lease settings for generated credentials.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azuresql/config/lease`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">ttl</span>
        <span class="param-flags">required</span>
        The ttl value provided as a string duration
        with time suffix. Hour is the largest suffix.
      </li>
      <li>
        <span class="param">ttl_max</span>
        <span class="param-flags">required</span>
        The maximum ttl value provided as a string duration
        with time suffix. Hour is the largest suffix.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azuresql/roles/
#### POST

<dl class="api">
  <dt>Description</dt>
  <dd>
    Creates or updates the role definition.
  </dd>

  <dt>Method</dt>
  <dd>POST</dd>

  <dt>URL</dt>
  <dd>`/azuresql/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">sql</span>
        <span class="param-flags">required</span>
        The SQL statements executed to create and configure the role.
        Must be semi-colon separated. User must be created in contained
        database user mode. The '{{name}}' and '{{password}}'
        values will be substituted.
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
  <dd>`/azuresql/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "data": {
        "sql": "CREATE USER..."
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
      "keys": ["dev", "prod"]
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
  <dd>`/azuresql/roles/<name>`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>
    A `204` response code.
  </dd>
</dl>

### /azuresql/creds/
#### GET

<dl class="api">
  <dt>Description</dt>
  <dd>
    Generates a new set of dynamic credentials based on the named role.
    Will also create a firewall rule for the (optional) given ip.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/azuresql/creds/<name>[/<ip>]`</dd>

  <dt>Parameters</dt>
  <dd>
     None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "data": {
        "fwrule":   "0b9ebcb5-801c-d85c-9aa1-e02e25558b91-112.22.25.32"
        "username": "root-a147d529-e7d6-4a16-8930-4c3e72170b19",
        "password": "ee202d0d-e4fd-4410-8d14-2a78c5c8cb76"
      }
    }
    ```

  </dd>
</dl>
