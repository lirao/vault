package azurestorage

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/logical"
	logicaltest "github.com/hashicorp/vault/logical/testing"
	"github.com/mitchellh/mapstructure"
)

func TestBackend_basic(t *testing.T) {
	b, _ := Factory(logical.TestBackendConfig())

	logicaltest.Test(t, logicaltest.TestCase{
		AcceptanceTest: true,
		PreCheck:       func() { testAccPreCheck(t) },
		Backend:        b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t),
			testAccStepRole(t),
			testAccStepReadVerifyToken(t, "web"),
		},
	})
}

func TestBackend_roleCrud(t *testing.T) {
	b := Backend()

	logicaltest.Test(t, logicaltest.TestCase{
		AcceptanceTest: true,
		PreCheck:       func() { testAccPreCheck(t) },
		Backend:        b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t),
			testAccStepRole(t),
			testAccStepReadRole(t, "web", os.Getenv("AZURE_STORAGE_POLICY"), 0),
			testAccStepDeleteRole(t, "web"),
			testAccStepReadRole(t, "web", "", 0),
		},
	})
}

func TestBackend_roleLeaseRead(t *testing.T) {
	b := Backend()

	logicaltest.Test(t, logicaltest.TestCase{
		AcceptanceTest: true,
		PreCheck:       func() { testAccPreCheck(t) },
		Backend:        b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t),
			testAccStepRoleLease(t, "30m"),
			testAccStepWriteLease(t),
			testAccStepReadRole(t, "web", os.Getenv("AZURE_STORAGE_POLICY"), 30*time.Minute),
			testAccStepReadLease(t),
		},
	})
}

func TestBackend_leaseWriteRead(t *testing.T) {
	b := Backend()

	logicaltest.Test(t, logicaltest.TestCase{
		AcceptanceTest: true,
		PreCheck:       func() { testAccPreCheck(t) },
		Backend:        b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t),
			testAccStepWriteLease(t),
			testAccStepReadLease(t),
		},
	})
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AZURE_STORAGE_RESNAME"); v == "" {
		t.Fatal("AZURE_STORAGE_RESNAME must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_NAMESPACE"); v == "" {
		t.Fatal("AZURE_STORAGE_NAMESPACE must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_POLICY"); v == "" {
		t.Fatal("AZURE_STORAGE_POLICY must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_KEY"); v == "" {
		t.Fatal("AZURE_STORAGE_KEY must be set for acceptance tests")
	}
}

func testAccStepConfig(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "config/resource",
		Data: map[string]interface{}{
			"name":      os.Getenv("AZURE_STORAGE_RESNAME"),
			"namespace": os.Getenv("AZURE_STORAGE_NAMESPACE"),
		},
	}
}

func testAccStepRole(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/web",
		Data: map[string]interface{}{
			"sas_policy_name": os.Getenv("AZURE_STORAGE_POLICY"),
			"sas_policy_key":  os.Getenv("AZURE_STORAGE_KEY"),
		},
	}
}

func testAccStepRoleLease(t *testing.T, ttl string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/web",
		Data: map[string]interface{}{
			"sas_policy_name": os.Getenv("AZURE_STORAGE_POLICY"),
			"sas_policy_key":  os.Getenv("AZURE_STORAGE_KEY"),
			"ttl":             ttl,
		},
	}
}

func testAccStepDeleteRole(t *testing.T, n string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      "roles/" + n,
	}
}

func testAccStepReadVerifyToken(t *testing.T, name string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "token/" + name,
		Check: func(resp *logical.Response) error {
			var d struct {
				Policy string `mapstructure:"policy_name"`
				Token  string `mapstructure:"token"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}
			log.Printf("[WARN] Generated token: %v", d)

			//Use HTTP POST REST API to verify this
			url := fmt.Sprintf("https://%s.servicebus.windows.net/%s/messages", os.Getenv("AZURE_STORAGE_NAMESPACE"), os.Getenv("AZURE_STORAGE_RESNAME"))

			client := &http.Client{}

			httpreq, err := http.NewRequest("POST", url, strings.NewReader("{}"))
			httpreq.Header.Add("Content-Type", "application/json")
			httpreq.Header.Add("ContentType", "application/atom+xml;type=entry;charset=utf-8")
			httpreq.Header.Add("Authorization", d.Token)
			httpresp, err := client.Do(httpreq)

			if err != nil {
				return err
			}
			if httpresp.StatusCode != 201 {
				return fmt.Errorf("[ERROR] Verification of SAS token failed with %s: %v", url, httpresp)
			}
			return nil
		},
	}
}

func testAccStepReadRole(t *testing.T, name, policy string, ttl time.Duration) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "roles/" + name,
		Check: func(resp *logical.Response) error {
			if resp == nil {
				if policy == "" {
					return nil
				}
				return fmt.Errorf("bad: %#v", resp)
			}

			var d struct {
				Policy string        `mapstructure:"sas_policy_name"`
				TTL    time.Duration `mapstructure:"ttl"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}

			if d.Policy != policy || (ttl > 0 && d.TTL != ttl) {
				return fmt.Errorf("bad: %#v", resp)
			}

			return nil
		},
	}
}

func testAccStepWriteLease(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "config/lease",
		Data: map[string]interface{}{
			"ttl": "1h5m",
		},
	}
}

func testAccStepReadLease(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "config/lease",
		Check: func(resp *logical.Response) error {
			if resp.Data["ttl"] != "1h5m0s" {
				return fmt.Errorf("bad: %#v", resp)
			}

			return nil
		},
	}
}
