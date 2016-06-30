package azurestorage

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathConfigAccount(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config/account",
		Fields: map[string]*framework.FieldSchema{
			"account_name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Name",
			},
			"account_key": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Storage Account Key",
			},
			"base_url": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     storage.DefaultBaseURL,
				Description: fmt.Sprintf("(Optional) Base URL of blob service. Defaults to %v", storage.DefaultBaseURL),
			},
			"api_version": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     storage.DefaultAPIVersion,
				Description: fmt.Sprintf("(Optional) Azure API version used. Defaults to %v", storage.DefaultAPIVersion),
			},
			"use_https": &framework.FieldSchema{
				Type:        framework.TypeBool,
				Default:     true,
				Description: "(Optional) Whether HTTPS is used. Defaults to true",
			},
			"verify": &framework.FieldSchema{
				Type:        framework.TypeBool,
				Default:     true,
				Description: "(Optional) If set, the Blob Account credentials are verified. Defaults to true",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathAccountWrite,
		},

		HelpSynopsis:    pathConfigAccountHelpSyn,
		HelpDescription: pathConfigAccountHelpDesc,
	}
}

func (b *backend) pathAccountWrite(
	req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("account_name").(string)
	key := data.Get("account_key").(string)
	baseURL := data.Get("base_url").(string)
	apiVer := data.Get("api_version").(string)
	https := data.Get("use_https").(bool)

	// verify
	verify := data.Get("verify").(bool)
	if verify {
		// Verify the credentials
		client, err := storage.NewClient(name, key, baseURL, apiVer, https)
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf(
				"error validating account info: %s", err)), nil
		}
		containers, err := client.GetBlobService().ListContainers(storage.ListContainersParameters{})
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf(
				"error validating connection info: %s", err)), nil
		}
		log.Print(containers)
	}

	// Store it
	entry, err := logical.StorageEntryJSON("config/account", accountConfig{
		Name:    name,
		Key:     key,
		BaseURL: baseURL,
		APIVer:  apiVer,
		HTTPS:   https,
	})

	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type accountConfig struct {
	Name    string `json:"account_name"`
	Key     string `json:"account_key"`
	BaseURL string `json:"base_url"`
	APIVer  string `json:"api_version"`
	HTTPS   bool   `json:"use_https"`
}

const pathConfigAccountHelpSyn = `
Configures the Blob Storage Account to connect to.
`

const pathConfigAccountHelpDesc = `
Configures the Blob Storage Account to connect to. Only account_name and account_key are required fields.
`
