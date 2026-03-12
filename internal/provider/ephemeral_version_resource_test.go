package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// protoV6ProviderFactories wires the provider into the test framework.
var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ephemeralversion": providerserver.NewProtocol6WithError(New()),
}

// uuidRegexp matches a standard UUID v4.
var uuidRegexp = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ── md5Hex unit tests ────────────────────────────────────────────────────────

func TestMd5Hex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"terraform", "1b1ed905d54c18e3dd8828986c14be17"},
	}

	for _, tt := range tests {
		got := md5Hex(tt.input)
		if got != tt.expected {
			t.Errorf("md5Hex(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// ── ephemeralversion_from acceptance tests ───────────────────────────────────

func fromConfig(value string) string {
	return fmt.Sprintf(`
resource "ephemeralversion_from" "test" {
  value = %q
}
`, value)
}

// TestEphemeralversionFrom_create verifies that after apply:
//   - id is a UUID
//   - version equals md5(value)
func TestEphemeralversionFrom_create(t *testing.T) {
	const value = "my-secret"
	expectedVersion := md5Hex(value)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fromConfig(value),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("ephemeralversion_from.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(expectedVersion)),
					statecheck.ExpectKnownValue("ephemeralversion_from.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegexp)),
				},
			},
		},
	})
}

// TestEphemeralversionFrom_idStableOnUpdate verifies that the UUID id does not
// change when value is updated, but version is recalculated.
func TestEphemeralversionFrom_idStableOnUpdate(t *testing.T) {
	const value1 = "first-secret"
	const value2 = "second-secret"

	var firstID string

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create — capture the id.
			{
				Config: fromConfig(value1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("ephemeralversion_from.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(md5Hex(value1))),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("ephemeralversion_from.test", "id",
						func(v string) error {
							if !uuidRegexp.MatchString(v) {
								return fmt.Errorf("id %q is not a UUID", v)
							}
							firstID = v
							return nil
						}),
				),
			},
			// Step 2: update value — id must be the same UUID, version must change.
			{
				Config: fromConfig(value2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("ephemeralversion_from.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(md5Hex(value2))),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("ephemeralversion_from.test", "id",
						func(v string) error {
							if v != firstID {
								return fmt.Errorf("id changed: was %q, now %q", firstID, v)
							}
							return nil
						}),
				),
			},
		},
	})
}

// TestEphemeralversionFrom_valueNotInState verifies that the write-only value
// is never present in state after apply.
func TestEphemeralversionFrom_valueNotInState(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fromConfig("super-secret"),
				Check:  resource.TestCheckNoResourceAttr("ephemeralversion_from.test", "value"),
			},
		},
	})
}

// ── ephemeralversion_from_map acceptance tests ───────────────────────────────

func fromMapConfig(secrets map[string]string) string {
	pairs := ""
	for k, v := range secrets {
		pairs += fmt.Sprintf("    %s = %q\n", k, v)
	}
	return fmt.Sprintf(`
resource "ephemeralversion_from_map" "test" {
  values = {
%s  }
}
`, pairs)
}

// TestEphemeralversionFromMap_create verifies that after apply each key in
// versions equals md5 of the corresponding input value.
func TestEphemeralversionFromMap_create(t *testing.T) {
	secrets := map[string]string{
		"db_password": "hunter2",
		"api_key":     "s3cr3t",
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fromMapConfig(secrets),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("ephemeralversion_from_map.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegexp)),
					statecheck.ExpectKnownValue("ephemeralversion_from_map.test",
						tfjsonpath.New("versions"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"db_password": knownvalue.StringExact(md5Hex("hunter2")),
							"api_key":     knownvalue.StringExact(md5Hex("s3cr3t")),
						})),
				},
			},
		},
	})
}

// TestEphemeralversionFromMap_idStableOnUpdate verifies UUID id is stable when
// values change, and versions are recalculated.
func TestEphemeralversionFromMap_idStableOnUpdate(t *testing.T) {
	v1 := map[string]string{"key": "value1"}
	v2 := map[string]string{"key": "value2"}

	var firstID string

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fromMapConfig(v1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("ephemeralversion_from_map.test", "id",
						func(v string) error {
							if !uuidRegexp.MatchString(v) {
								return fmt.Errorf("id %q is not a UUID", v)
							}
							firstID = v
							return nil
						}),
					resource.TestCheckResourceAttr("ephemeralversion_from_map.test",
						"versions.key", md5Hex("value1")),
				),
			},
			{
				Config: fromMapConfig(v2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("ephemeralversion_from_map.test", "id",
						func(v string) error {
							if v != firstID {
								return fmt.Errorf("id changed: was %q, now %q", firstID, v)
							}
							return nil
						}),
					resource.TestCheckResourceAttr("ephemeralversion_from_map.test",
						"versions.key", md5Hex("value2")),
				),
			},
		},
	})
}

// TestEphemeralversionFromMap_valuesNotInState verifies that write-only values
// are never present in state after apply.
func TestEphemeralversionFromMap_valuesNotInState(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fromMapConfig(map[string]string{"secret": "topsecret"}),
				Check:  resource.TestCheckNoResourceAttr("ephemeralversion_from_map.test", "values"),
			},
		},
	})
}
