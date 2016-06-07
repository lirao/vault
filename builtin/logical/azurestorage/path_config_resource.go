package azurestorage

import (
	"fmt"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/Azure/azure-sdk-for-go/blob/master/storage"
)

func pathConfigResource(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config/resource",
		Fields: map[string]*framework.FieldSchema{
			"account_name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Name",
			},
			"account_key": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Key",
			},
			"blob_service_base_url": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Name",
			},
			"account_key": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Key",
			},
		},
accountName, accountKey, blobServiceBaseURL, apiVer
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathResourceWrite,
		},

		HelpSynopsis:    pathConfigResourceHelpSyn,
		HelpDescription: pathConfigResourceHelpDesc,
	}
}

func (b *backend) pathResourceWrite(
	req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	namespace := data.Get("namespace").(string)
	uri := fmt.Sprintf("https://%s.servicebus.windows.net/%s", namespace, name)

	// Store it
	entry, err := logical.StorageEntryJSON("config/resource", resourceConfig{
		ResourceURI: uri,
	})
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type resourceConfig struct {
	ResourceURI string `json:"uri"`
}

const pathConfigResourceHelpSyn = `
Configure the uri for Service Bus SAS tokens.
`

const pathConfigResourceHelpDesc = `
Configures a Service Bus resource that you need SAS tokens for.
`
