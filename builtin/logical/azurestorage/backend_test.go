package azurestorage

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
			testAccStepRoleBlob(t),
			testAccStepReadVerifyTokenBlob(t),
		},
	})
}

func TestBackend_container(t *testing.T) {
	b, _ := Factory(logical.TestBackendConfig())

	logicaltest.Test(t, logicaltest.TestCase{
		AcceptanceTest: true,
		PreCheck:       func() { testAccPreCheck(t) },
		Backend:        b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t),
			testAccStepRoleContainer(t),
			testAccStepReadVerifyTokenContainer(t),
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
			testAccStepRoleBlob(t),
			testAccStepReadRole(t, "blob", os.Getenv("AZURE_STORAGE_CONTAINER"), 0),
			testAccStepDeleteRole(t, "blob"),
			testAccStepReadRole(t, "blob", "", 0),
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
			testAccStepReadRole(t, "web", os.Getenv("AZURE_STORAGE_CONTAINER"), 30*time.Minute),
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
	if v := os.Getenv("AZURE_STORAGE_ACCESS_KEY"); v == "" {
		t.Fatal("AZURE_STORAGE_ACCESS_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_ACCOUNT"); v == "" {
		t.Fatal("AZURE_STORAGE_ACCOUNT must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_CONTAINER"); v == "" {
		t.Fatal("AZURE_STORAGE_CONTAINER must be set for acceptance tests")
	}
	if v := os.Getenv("AZURE_STORAGE_BLOB"); v == "" {
		t.Fatal("AZURE_STORAGE_BLOB must be set for acceptance tests")
	}
}

func testAccStepConfig(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "config/account",
		Data: map[string]interface{}{
			"account_name": os.Getenv("AZURE_STORAGE_ACCOUNT"),
			"account_key":  os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		},
	}
}

func testAccStepRoleBlob(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/blob",
		Data: map[string]interface{}{
			"container":   os.Getenv("AZURE_STORAGE_CONTAINER"),
			"blob":        os.Getenv("AZURE_STORAGE_BLOB"),
			"permissions": "r",
			"ttl":         "15m",
		},
	}
}

func testAccStepRoleLease(t *testing.T, ttl string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/web",
		Data: map[string]interface{}{
			"container":   os.Getenv("AZURE_STORAGE_CONTAINER"),
			"blob":        os.Getenv("AZURE_STORAGE_BLOB"),
			"permissions": "r",
			"ttl":         ttl,
		},
	}
}

func testAccStepRoleContainer(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/container",
		Data: map[string]interface{}{
			"container":   os.Getenv("AZURE_STORAGE_CONTAINER"),
			"permissions": "rl",
			"ttl":         "15m",
		},
	}
}

func testAccStepDeleteRole(t *testing.T, n string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      "roles/" + n,
	}
}

func testAccStepReadVerifyTokenBlob(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "token/blob",
		Check: func(resp *logical.Response) error {
			var d struct {
				URI string `mapstructure:"uri"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}
			log.Printf("[WARN] Generated URI: %v", d)

			httpresp, err := http.Get(d.URI)

			if err != nil {
				return err
			}
			if httpresp.StatusCode != 200 {
				return fmt.Errorf("[ERROR] Verification of SAS token (single blob) failed with %s: %v", d.URI, httpresp)
			}
			return nil
		},
	}
}

func testAccStepReadVerifyTokenContainer(t *testing.T) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "token/container",
		Check: func(resp *logical.Response) error {
			var d struct {
				URI string `mapstructure:"uri"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}
			log.Printf("[WARN] Generated URI: %v", d)

			url := fmt.Sprintf("%s&comp=list&restype=container", d.URI)

			httpresp, err := http.Get(url)

			if err != nil {
				return err
			}
			if httpresp.StatusCode != 200 {
				return fmt.Errorf("[ERROR] Verification of SAS token (container) failed with %s: %v", url, httpresp)
			}
			return nil
		},
	}
}

func testAccStepReadRole(t *testing.T, name, container string, ttl time.Duration) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "roles/" + name,
		Check: func(resp *logical.Response) error {
			if resp == nil {
				if container == "" {
					return nil
				}
				return fmt.Errorf("bad: %#v", resp)
			}

			var d struct {
				Container string        `mapstructure:"container"`
				TTL       time.Duration `mapstructure:"ttl"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}

			if d.Container != container || (ttl > 0 && d.TTL != ttl) {
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
