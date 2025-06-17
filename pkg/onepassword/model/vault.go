package model

import (
	"time"

	connect "github.com/1Password/connect-sdk-go/onepassword"
	sdk "github.com/1password/onepassword-sdk-go"
)

type Vault struct {
	ID        string
	CreatedAt time.Time
}

func (v *Vault) FromConnectVault(vault *connect.Vault) {
	v.ID = vault.ID
	v.CreatedAt = vault.CreatedAt
}

func (v *Vault) FromSDKVault(vault *sdk.VaultOverview) {
	v.ID = vault.ID
	v.CreatedAt = vault.CreatedAt
}
