package connect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/1Password/connect-sdk-go/onepassword"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaegerClientConfig "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/zipkin"
)

const (
	defaultUserAgent = "connect-sdk-go/0.0.1"
)

// Client Represents an available 1Password Connect API to connect to
type Client interface {
	GetVaults() ([]onepassword.Vault, error)
	GetItem(uuid string, vaultUUID string) (*onepassword.Item, error)
	GetItems(vaultUUID string) ([]onepassword.Item, error)
	GetItemByTitle(title string, vaultUUID string) (*onepassword.Item, error)
	CreateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	UpdateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	DeleteItem(item *onepassword.Item, vaultUUID string) error
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	envHostVariable  = "OP_CONNECT_HOST"
	envTokenVariable = "OP_CONNECT_TOKEN"
)

// NewClientFromEnvironment Returns a Secret Service client assuming that your
// jwt is set in the OP_TOKEN environment variable
func NewClientFromEnvironment() (Client, error) {
	host, found := os.LookupEnv(envHostVariable)
	if !found {
		return nil, fmt.Errorf("There is no hostname available in the %q variable", envHostVariable)
	}

	token, found := os.LookupEnv(envTokenVariable)
	if !found {
		return nil, fmt.Errorf("There is no token available in the %q variable", envTokenVariable)
	}

	return NewClient(host, token), nil
}

// NewClient Returns a Secret Service client for a given url and jwt
func NewClient(url string, token string) Client {
	return NewClientWithUserAgent(url, token, defaultUserAgent)
}

// NewClientWithUserAgent Returns a Secret Service client for a given url and jwt and identifies with userAgent
func NewClientWithUserAgent(url string, token string, userAgent string) Client {
	if !opentracing.IsGlobalTracerRegistered() {
		cfg := jaegerClientConfig.Configuration{}
		zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
		cfg.InitGlobalTracer(
			userAgent,
			jaegerClientConfig.Injector(opentracing.HTTPHeaders, zipkinPropagator),
			jaegerClientConfig.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
			jaegerClientConfig.ZipkinSharedRPCSpan(true),
		)
	}

	return &restClient{
		URL:   url,
		Token: token,

		userAgent: userAgent,
		tracer:    opentracing.GlobalTracer(),

		client: http.DefaultClient,
	}
}

type restClient struct {
	URL       string
	Token     string
	userAgent string
	tracer    opentracing.Tracer
	client    httpClient
}

// GetVaults Get a list of all available vaults
func (rs *restClient) GetVaults() ([]onepassword.Vault, error) {
	span := rs.tracer.StartSpan("GetVaults")
	defer span.Finish()

	vaultURL := fmt.Sprintf("/v1/vaults")
	request, err := rs.buildRequest(http.MethodGet, vaultURL, http.NoBody, span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to retrieve vaults. Receieved %q for %q", response.Status, vaultURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	vaults := []onepassword.Vault{}
	if err := json.Unmarshal(body, &vaults); err != nil {
		return nil, err
	}

	return vaults, nil
}

// GetItem Get a specific Item from the 1Password Connect API
func (rs *restClient) GetItem(uuid string, vaultUUID string) (*onepassword.Item, error) {
	span := rs.tracer.StartSpan("GetItem")
	defer span.Finish()

	itemURL := fmt.Sprintf("/v1/vaults/%s/items/%s", vaultUUID, uuid)
	request, err := rs.buildRequest(http.MethodGet, itemURL, http.NoBody, span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to retrieve item. Receieved %q for %q", response.Status, itemURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	item := onepassword.Item{}
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (rs *restClient) GetItemByTitle(title string, vaultUUID string) (*onepassword.Item, error) {
	span := rs.tracer.StartSpan("GetItemByTitle")
	defer span.Finish()

	filter := url.QueryEscape(fmt.Sprintf("title eq \"%s\"", title))
	itemURL := fmt.Sprintf("/v1/vaults/%s/items?filter=%s", vaultUUID, filter)
	request, err := rs.buildRequest(http.MethodGet, itemURL, http.NoBody, span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to retrieve item. Receieved %q for %q", response.Status, itemURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	items := []onepassword.Item{}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	if len(items) != 1 {
		return nil, fmt.Errorf("Found %d item(s) in vault %q with title %q", len(items), vaultUUID, title)
	}

	return rs.GetItem(items[0].ID, items[0].Vault.ID)
}

func (rs *restClient) GetItems(vaultUUID string) ([]onepassword.Item, error) {
	span := rs.tracer.StartSpan("GetItems")
	defer span.Finish()

	itemURL := fmt.Sprintf("/v1/vaults/%s/items", vaultUUID)
	request, err := rs.buildRequest(http.MethodGet, itemURL, http.NoBody, span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to retrieve items. Receieved %q for %q", response.Status, itemURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	items := []onepassword.Item{}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	return items, nil
}

// CreateItem Create a new item in a specified vault
func (rs *restClient) CreateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	span := rs.tracer.StartSpan("CreateItem")
	defer span.Finish()

	itemURL := fmt.Sprintf("/v1/vaults/%s/items", vaultUUID)
	itemBody, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	request, err := rs.buildRequest(http.MethodPost, itemURL, bytes.NewBuffer(itemBody), span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to create item. Receieved %q for %q", response.Status, itemURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	newItem := onepassword.Item{}
	if err := json.Unmarshal(body, &newItem); err != nil {
		return nil, err
	}

	return &newItem, nil
}

// UpdateItem Update a new item in a specified vault
func (rs *restClient) UpdateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	span := rs.tracer.StartSpan("UpdateItem")
	defer span.Finish()

	itemURL := fmt.Sprintf("/v1/vaults/%s/items/%s", item.Vault.ID, item.ID)
	itemBody, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	request, err := rs.buildRequest(http.MethodPut, itemURL, bytes.NewBuffer(itemBody), span)
	if err != nil {
		return nil, err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to update item. Receieved %q for %q", response.Status, itemURL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	newItem := onepassword.Item{}
	if err := json.Unmarshal(body, &newItem); err != nil {
		return nil, err
	}

	return &newItem, nil
}

// DeleteItem Delete a new item in a specified vault
func (rs *restClient) DeleteItem(item *onepassword.Item, vaultUUID string) error {
	span := rs.tracer.StartSpan("DeleteItem")
	defer span.Finish()

	itemURL := fmt.Sprintf("/v1/vaults/%s/items/%s", item.Vault.ID, item.ID)
	request, err := rs.buildRequest(http.MethodDelete, itemURL, http.NoBody, span)
	if err != nil {
		return err
	}

	response, err := rs.client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Unable to retrieve item. Receieved %q for %q", response.Status, itemURL)
	}

	return nil
}

func (rs *restClient) buildRequest(method string, path string, body io.Reader, span opentracing.Span) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", rs.URL, path)

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rs.Token))
	request.Header.Set("User-Agent", rs.userAgent)

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, path)
	ext.HTTPMethod.Set(span, method)

	rs.tracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(request.Header))

	return request, nil
}
