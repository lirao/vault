package azureservicebus

import (
	"strings"
	"sync"

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
			pathConfigResource(&b),
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

	lock sync.Mutex
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

// ResourceConfig returns the Event Hub resource configuration
func (b *backend) ResourceConfig(s logical.Storage) (*resourceConfig, error) {
	entry, err := s.Get("config/resource")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result resourceConfig
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

const backendHelp = `
The Azure Service Bus SAS Token backend generates SAS tokens for Service Bus 
resources, which can include Service Bus relays, queues, topics, and Event Hubs.

Explaination and usage:
https://azure.microsoft.com/en-us/documentation/articles/service-bus-sas-overview/

After mounting this backend, configure it using the endpoints within
the "config/" path.

Not to be confused with Azure Storage SAS URIs, which is supported by the 
azurestorage backend.
`
