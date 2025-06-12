package mock

import (
	"context"

	"github.com/stretchr/testify/mock"

	sdk "github.com/1password/onepassword-sdk-go"
)

type VaultAPIMock struct {
	mock.Mock
}

func (v *VaultAPIMock) List(ctx context.Context) ([]sdk.VaultOverview, error) {
	args := v.Called(ctx)
	return args.Get(0).([]sdk.VaultOverview), args.Error(1)
}

type ItemAPIMock struct {
	mock.Mock
	FilesAPI sdk.ItemsFilesAPI
}

func (i *ItemAPIMock) Create(ctx context.Context, params sdk.ItemCreateParams) (sdk.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (i *ItemAPIMock) Get(ctx context.Context, vaultID string, itemID string) (sdk.Item, error) {
	args := i.Called(ctx, vaultID, itemID)
	return args.Get(0).(sdk.Item), args.Error(1)
}

func (i *ItemAPIMock) Put(ctx context.Context, item sdk.Item) (sdk.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (i *ItemAPIMock) Delete(ctx context.Context, vaultID string, itemID string) error {
	//TODO implement me
	panic("implement me")
}

func (i *ItemAPIMock) Archive(ctx context.Context, vaultID string, itemID string) error {
	//TODO implement me
	panic("implement me")
}

func (i *ItemAPIMock) List(ctx context.Context, vaultID string, filters ...sdk.ItemListFilter) ([]sdk.ItemOverview, error) {
	args := i.Called(ctx, vaultID, filters)
	return args.Get(0).([]sdk.ItemOverview), args.Error(1)
}

func (i *ItemAPIMock) Shares() sdk.ItemsSharesAPI {
	//TODO implement me
	panic("implement me")
}

func (i *ItemAPIMock) Files() sdk.ItemsFilesAPI {
	return i.FilesAPI
}

type FileAPIMock struct {
	mock.Mock
}

func (f *FileAPIMock) Attach(ctx context.Context, item sdk.Item, fileParams sdk.FileCreateParams) (sdk.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FileAPIMock) Delete(ctx context.Context, item sdk.Item, sectionID string, fieldID string) (sdk.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FileAPIMock) ReplaceDocument(ctx context.Context, item sdk.Item, docParams sdk.DocumentCreateParams) (sdk.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FileAPIMock) Read(ctx context.Context, vaultID, itemID string, attributes sdk.FileAttributes) ([]byte, error) {
	args := f.Called(ctx, vaultID, itemID, attributes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
