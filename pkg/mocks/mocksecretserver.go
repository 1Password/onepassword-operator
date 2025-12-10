package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

type TestClient struct {
	mock.Mock
}

func (tc *TestClient) GetItemByID(ctx context.Context, vaultID, itemID string) (*model.Item, error) {
	args := tc.Called(vaultID, itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Item), args.Error(1)
}

func (tc *TestClient) GetItemsByTitle(ctx context.Context, vaultID, itemTitle string) ([]model.Item, error) {
	args := tc.Called(vaultID, itemTitle)
	return args.Get(0).([]model.Item), args.Error(1)
}

func (tc *TestClient) GetFileContent(ctx context.Context, vaultID, itemID, fileID string) ([]byte, error) {
	args := tc.Called(vaultID, itemID, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (tc *TestClient) GetVaultsByTitle(ctx context.Context, title string) ([]model.Vault, error) {
	args := tc.Called(title)
	return args.Get(0).([]model.Vault), args.Error(1)
}
