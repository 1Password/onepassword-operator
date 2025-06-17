package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	connect "github.com/1Password/connect-sdk-go/onepassword"
	sdk "github.com/1password/onepassword-sdk-go"
)

func TestVault_FromConnectVault(t *testing.T) {
	connectVault := &connect.Vault{
		ID:        "test-id",
		CreatedAt: time.Now(),
	}

	vault := &Vault{}
	vault.FromConnectVault(connectVault)

	require.Equal(t, connectVault.ID, vault.ID)
	require.Equal(t, connectVault.CreatedAt, vault.CreatedAt)
}

func TestVault_FromSDKVault(t *testing.T) {
	sdkVault := &sdk.VaultOverview{
		ID:        "test-id",
		CreatedAt: time.Now(),
	}

	vault := &Vault{}
	vault.FromSDKVault(sdkVault)

	require.Equal(t, sdkVault.ID, vault.ID)
	require.Equal(t, sdkVault.CreatedAt, vault.CreatedAt)
}
