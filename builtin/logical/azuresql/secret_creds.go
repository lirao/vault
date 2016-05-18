package azuresql

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const SecretCredsType = "creds"

func secretCreds(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretCredsType,
		Fields: map[string]*framework.FieldSchema{
			"username": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Username",
			},

			"password": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Password",
			},
		},

		Renew:  b.secretCredsRenew,
		Revoke: b.secretCredsRevoke,
	}
}

func (b *backend) secretCredsRenew(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	// Get the lease information
	leaseConfig, err := b.LeaseConfig(req.Storage)
	if err != nil {
		return nil, err
	}
	if leaseConfig == nil {
		leaseConfig = &configLease{}
	}

	f := framework.LeaseExtend(leaseConfig.TTL, leaseConfig.TTLMax, b.System())
	return f(req, d)
}

func (b *backend) secretCredsRevoke(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	// Get the username from the internal data
	usernameRaw, ok := req.Secret.InternalData["username"]
	if !ok {
		return nil, fmt.Errorf("secret is missing username internal data")
	}

	username, ok := usernameRaw.(string)

	// Get our connection
	db, err := b.DB(req.Storage)
	if err != nil {
		return nil, err
	}

	// First disable server login
	disableStmt, err := db.Prepare(fmt.Sprintf("REVOKE CONNECT FROM [%s];", username))
	if err != nil {
		return nil, err
	}
	defer disableStmt.Close()
	if _, err := disableStmt.Exec(); err != nil {
		return nil, err
	}

	// Query for sessions for the login so that we can kill any outstanding
	// sessions.  There cannot be any active sessions before we drop the user
	sessionStmt, err := db.Prepare(fmt.Sprintf(
		"SELECT session_id FROM sys.dm_exec_sessions WHERE login_name = '%s';", username))
	if err != nil {
		return nil, err
	}
	defer sessionStmt.Close()

	sessionRows, err := sessionStmt.Query()
	if err != nil {
		return nil, err
	}
	defer sessionRows.Close()

	var revokeStmts []string
	for sessionRows.Next() {
		var sessionID int
		err = sessionRows.Scan(&sessionID)
		if err != nil {
			return nil, err
		}
		revokeStmts = append(revokeStmts, fmt.Sprintf("KILL %d;", sessionID))
	}

	// Drop this user
	dropUserSQL := "DROP USER IF EXISTS [%s]"
	stmt, err := db.Prepare(fmt.Sprintf(dropUserSQL, username))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(); err != nil {
		return nil, err
	}

	//Delete the firewall rule from Azure
	fwrule, ok := req.Secret.InternalData["fwrule"].(string)
	if ok && len(fwrule) > 0 {
		client, err := b.AzureClient(req.Storage)
		if err != nil {
			return nil, err
		}
		err = client.DeleteFirewallRule(b.server, fwrule)
		if err != nil {
			if strings.Contains(err.Error(), "To continue, specify a valid resource name.") {
				b.Logger().Printf("Firewall Rule %s already deleted", fwrule)
			} else {
				return nil, err
			}
		}
	}

	return nil, nil
}
