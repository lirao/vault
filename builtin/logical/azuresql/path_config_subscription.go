package azuresql

import (
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathConfigSubscription(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config/subscription",
		Fields: map[string]*framework.FieldSchema{
			"subscription_id": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Azure subscription ID",
			},
			"management_cert": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Path to the management certificate PEM file",
			},
			"server": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Server name",
			},
			"database": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Database name",
			},
			"publish_settings": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Path to .publisSettings file that has subscription creds",
			},
			"verify": &framework.FieldSchema{
				Type:        framework.TypeBool,
				Default:     true,
				Description: "If set, subscription and certificate are verified by connecting to Azure",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathSubscriptionWrite,
		},

		HelpSynopsis:    pathConfigSubscriptionHelpSyn,
		HelpDescription: pathConfigSubscriptionHelpDesc,
	}
}

func (b *backend) pathSubscriptionWrite(
	req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	publishSettings := data.Get("publish_settings").(string)
	subscriptionID := data.Get("subscription_id").(string)
	managementCert := data.Get("management_cert").(string)
	server := data.Get("server").(string)
	database := data.Get("database").(string)

	// Don't check the subscription if verification is disabled
	verifyConnection := data.Get("verify").(bool)
	if verifyConnection {
		// Use the Azure Go Client
		var client management.Client
		var err error
		if len(publishSettings) > 0 {
			client, err = management.ClientFromPublishSettingsFile(publishSettings, subscriptionID)
			if err != nil {
				return nil, err
			}
		} else {
			cert, err := ioutil.ReadFile(managementCert)
			if err != nil {
				return nil, err
			}
			client, err = management.NewClient(subscriptionID, cert)
			if err != nil {
				return nil, err
			}
		}
		dbclient := sql.NewClient(client)
		_, err = dbclient.GetDatabase(server, database)
		if err != nil {
			return nil, err
		}
	}

	// Store it
	entry, err := logical.StorageEntryJSON("config/subscription", subscriptionConfig{
		SubscriptionID: subscriptionID,
		ManagementCert: managementCert,
		Server:         server,
		Database:       database,
	})

	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type subscriptionConfig struct {
	SubscriptionID string `json:"subscription_id"`
	ManagementCert string `json:"management_cert"`
	Server         string `json:"server"`
	Database       string `json:"database"`
}

const pathConfigSubscriptionHelpSyn = `
Configure the subscription and connection details to talk to Azure Sql Server.
`

const pathConfigSubscriptionHelpDesc = `
This path configures the subscription credentials of an the Azure subscription that the Azure SQL server belongs to.

When configuring the subscription, the backend will verify its validity.
If the subscription is not available when setting the connection string, set the
"verify_connection" option to false.
`
