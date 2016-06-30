package azurestorage

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/storage"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func Factory(conf *logical.BackendConfig) (logical.Backend, error) {
	return Backend().Setup(conf)
}

func Backend() *framework.Backend {
	var b backend
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),

		Paths: []*framework.Path{
			pathConfigAccount(&b),
			pathConfigLease(&b),
			pathListRoles(&b),
			pathRoles(&b),
			pathToken(&b),
		},

		Secrets: []*framework.Secret{
			secretToken(&b),
		},
	}

	return b.Backend
}

type backend struct {
	*framework.Backend
	client *storage.Client
	lock   sync.Mutex
}

func (b *backend) Account(s logical.Storage) (*accountConfig, error) {
	//Init the client
	entry, err := s.Get("config/account")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil,
			fmt.Errorf("configure the Storage Account information with config/account first")
	}

	var accConfig accountConfig
	if err := entry.DecodeJSON(&accConfig); err != nil {
		return nil, err
	}
	return &accConfig, nil
}

func (b *backend) StorageClient(s logical.Storage) (*storage.Client, error) {
	if b.client == nil {
		//Init the client
		accConfig, err := b.Account(s)
		if err != nil {
			return nil, err
		}

		client, err := storage.NewClient(accConfig.Name, accConfig.Key, accConfig.BaseURL, accConfig.APIVer, accConfig.HTTPS)
		if err != nil {
			return nil, err
		}
		b.client = &client
	}
	return b.client, nil
}

// LeaseConfig returns the lease configuration
func (b *backend) LeaseConfig(s logical.Storage) (*configLease, error) {
	entry, err := s.Get("config/lease")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result configLease
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

const backendHelp = `
The Azure Storage backend generates a SAS URI that grants restricted access 
to Azure Storage resources.

Explaination and usage:
https://azure.microsoft.com/en-us/documentation/articles/storage-dotnet-shared-access-signature-part-1/#examples-create-and-use-shared-access-signatures

After mounting this backend, configure it using the endpoints within 
the "config/" path.

Not to be confused with Azure Service Bus SAS tokens, which is supported by 
the azureservicebus backend. 

Only supports blob storage Service SAS (for now).
`
