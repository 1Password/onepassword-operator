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
		{
			"vaults/foo1/items/bar1/vaults/foo2/items/bar2",
			"foo1/items/bar1/vaults/foo2",
			"bar2",
			nil,
		},
		{
			"items/bar/vaults/foo",
			"",
			"",
			fmt.Errorf("\"items/bar/vaults/foo\" is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`"),
		},
		{
			"vaults/foo",
			"",
			"",
			fmt.Errorf("\"vaults/foo\" is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`"),
		},
		{
			"vaults/foo/items/bar/",
			"foo",
			"bar/",
			nil,
		},
		{
			"vaults/abc123-def456/items/xyz789-uvw012",
			"abc123-def456",
			"xyz789-uvw012",
			nil,
		},
		{
			"vaults/a/items/b",
			"a",
			"b",
			nil,
		},
		{
			"",
			"",
			"",
			fmt.Errorf("\"\" is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`"),
		},
		{
			"vaults//foo/items/bar",
			"/foo",
			"bar",
			nil,
		},
		{
			"vaults/foo/items/",
			"foo",
			"",
			nil,
		},
		{
			"vaults/items",
			"",
			"",
			fmt.Errorf("\"vaults/items\" is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`"),
		},
		{
			"vaults/foo bar/items/baz",
			"foo bar",
			"baz",
			nil,
		},
		{
			"vaults/日本/items/条目",
			"日本",
			"条目",
			nil,
		},
		{
			"prefix/vaults/foo/items/bar",
			"foo",
			"bar",
			nil,
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
