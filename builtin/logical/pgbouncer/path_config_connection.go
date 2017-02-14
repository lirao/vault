package pgbouncer

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	_ "github.com/lib/pq"
)

func pathConfigConnection(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config/connection",
		Fields: map[string]*framework.FieldSchema{
			"connection_urls": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "DB connection string to postgres",
			},
			"verify_connection": &framework.FieldSchema{
				Type:        framework.TypeBool,
				Default:     true,
				Description: `If set, connection_url is verified by actually connecting to the database`,
			},
			"max_open_connections": &framework.FieldSchema{
				Type: framework.TypeInt,
				Description: `Maximum number of open connections to the database;
a zero uses the default value of two and a
negative value means unlimited`,
			},

			// Implementation note:
			"max_idle_connections": &framework.FieldSchema{
				Type: framework.TypeInt,
				Description: `Maximum number of idle connections to the database;
a zero uses the value of max_open_connections
and a negative value disables idle connections.
If larger than max_open_connections it will be
reduced to the same size.`,
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathConnectionWrite,
		},

		HelpSynopsis:    pathConfigConnectionHelpSyn,
		HelpDescription: pathConfigConnectionHelpDesc,
	}
}

func (b *backend) pathConnectionWrite(
	req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	connValue := data.Get("value").(string)
	connURL := data.Get("connection_urls").(string)
	if connURL == "" {
		if connValue == "" {
			return logical.ErrorResponse("connection_urls parameter must be supplied with at least one connection string"), nil
		} else {
			connURL = connValue
		}
	}
	connStrings := strings.Split(connURL, `\n`)

	fmt.Print(connStrings)

	maxOpenConns := data.Get("max_open_connections").(int)
	if maxOpenConns == 0 {
		maxOpenConns = 2
	}

	maxIdleConns := data.Get("max_idle_connections").(int)
	if maxIdleConns == 0 {
		maxIdleConns = maxOpenConns
	}
	if maxIdleConns > maxOpenConns {
		maxIdleConns = maxOpenConns
	}

	// Don't check the connection_url if verification is disabled
	verifyConnection := data.Get("verify_connection").(bool)
	if verifyConnection {
		// Verify the string
		db, err := sql.Open("postgres", connURL)
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf(
				"Error validating connection info: %s", err)), nil
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			return logical.ErrorResponse(fmt.Sprintf(
				"Error validating connection info: %s", err)), nil
		}
	}

	// Store it
	entry, err := logical.StorageEntryJSON("config/connection", connectionConfig{
		ConnectionString:   connValue,
		ConnectionURL:      connURL,
		MaxOpenConnections: maxOpenConns,
		MaxIdleConnections: maxIdleConns,
	})
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	// Reset the DB connection
	b.ResetDB()

	return nil, nil
}

type connectionConfig struct {
	ConnectionURL string `json:"connection_url"`
	// Deprecate "value" in coming releases
	ConnectionString   string `json:"value"`
	MaxOpenConnections int    `json:"max_open_connections"`
	MaxIdleConnections int    `json:"max_idle_connections"`
}

const pathConfigConnectionHelpSyn = `
Configure the connection string to talk to PostgreSQL.
`

const pathConfigConnectionHelpDesc = `
This path configures the connection string used to connect to PostgreSQL.
The value of the string can be a URL, or a PG style string in the
format of "user=foo host=bar" etc.

The URL looks like:
"postgresql://user:pass@host:port/dbname"

When configuring the connection string, the backend will verify its validity.
`
