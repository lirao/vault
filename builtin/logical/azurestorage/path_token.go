package azurestorage

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathToken(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "token/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Name of the role.",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation: b.pathTokenRead,
		},

		HelpSynopsis:    pathTokenHelpSyn,
		HelpDescription: pathTokenHelpDesc,
	}
}

func (b *backend) pathTokenRead(
	req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	// Get the role
	role, err := b.Role(req.Storage, name)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse(fmt.Sprintf("unknown role: %s", name)), nil
	}

	ttl := role.TTL
	// Determine if we have a lease configuration
	if ttl == 0 {
		leaseConfig, err := b.LeaseConfig(req.Storage)
		if err != nil {
			return nil, err
		}
		if leaseConfig == nil {
			leaseConfig = &configLease{}
		}
		ttl = leaseConfig.TTL
	}

	expiry := time.Now().Add(ttl)

	client, err := b.StorageClient(req.Storage)
	if err != nil {
		return nil, err
	}
	blobcli := client.GetBlobService()
	uri, err := blobcli.GetBlobSASURI(role.Container, role.Blob, expiry, role.Permissions)
	if err != nil {
		return nil, err
	}

	// Return the secret. No data need to be saved in the secret itself
	resp := b.Secret(SecretTokenType).Response(map[string]interface{}{
		"blob":        role.Blob,
		"container":   role.Container,
		"permissions": role.Permissions,
		"uri":         uri,
	}, map[string]interface{}{})
	resp.Secret.TTL = ttl
	return resp, nil
}

const pathTokenHelpSyn = `
Request a SAS URI for a certain role.
`

const pathTokenHelpDesc = `
This path generates a SAS URI for a certain role. The
URI is generated on demand and will automatically 
expire when the lease is up.
`
