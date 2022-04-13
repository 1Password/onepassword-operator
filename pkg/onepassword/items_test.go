package onepassword

import (
	"fmt"
	"testing"
)

func TestParseVaultAndItemFromPath(t *testing.T) {
	cases := []struct {
		Path  string
		Vault string
		Item  string
		Error error
	}{
		{
			"vaults/foo/items/bar",
			"foo",
			"bar",
			nil,
		},
		{
			"vaults/foo/items/bar/baz",
			"foo",
			"bar/baz",
			nil,
		},
		{
			"vaults/foo/bar/items/baz",
			"foo/bar",
			"baz",
			nil,
		},
		{
			"foo/bar",
			"",
			"",
			fmt.Errorf("\"foo/bar\" is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`"),
		},
	}

	for _, c := range cases {
		vault, item, err := ParseVaultAndItemFromPath(c.Path)

		if err != c.Error && err.Error() != c.Error.Error() {
			t.Errorf("unexpected error %v: %v", err, c.Error)
		}

		if vault != c.Vault {
			t.Errorf("couldn't extract vault out of path %s: %s", c.Path, vault)
		}

		if item != c.Item {
			t.Errorf("couldn't extract item out of path %s: %s != %s", c.Path, item, c.Item)
		}

	}
}
