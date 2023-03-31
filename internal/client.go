package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/google/jsonapi"
	"golang.org/x/time/rate"
)

const BASE_URL = "https://api.imgix.com/api/v1/"

const (
	ImgixResourceSource string = "sources"
	ImgixResourcePurge  string = "purges"
	ImgixResourceReport string = "reports"
)

type ImgixSource struct {
	ID               string                `jsonapi:"primary,sources,omitempty" json:"id,omitempty"`
	Name             string                `jsonapi:"attr,name,omitempty" json:"name,omitempty"`
	Enabled          *bool                 `jsonapi:"attr,enabled,omitempty" json:"enabled,omitempty"`
	Deployment       ImgixSourceDeployment `jsonapi:"attr,deployment,omitempty" json:"deployment,omitempty"`
	DeploymentStatus string                `jsonapi:"attr,deployment_status,omitempty" json:"deployment_status,omitempty"`
	SecureURLToken   string                `jsonapi:"attr,secure_url_token,omitempty" json:"secure_url_token,omitempty"`
	DateDeployed     int                   `jsonapi:"attr,date_deployed,omitempty" json:"date_deployed,omitempty"`
}

type ImgixSourceDeployment struct {
	AllowsUpload          bool                   `jsonapi:"attr,allows_upload" json:"allows_upload,omitempty"`
	Annotation            string                 `jsonapi:"attr,annotation" json:"annotation,omitempty"`
	CacheTTLBehavior      string                 `jsonapi:"attr,cache_ttl_behavior" json:"cache_ttl_behavior,omitempty"`
	CacheTTLError         int                    `jsonapi:"attr,cache_ttl_error" json:"cache_ttl_error,omitempty"`
	CacheTTLValue         int                    `jsonapi:"attr,cache_ttl_value" json:"cache_ttl_value,omitempty"`
	CrossdomainXMLEnabled bool                   `jsonapi:"attr,crossdomain_xml_enabled" json:"crossdomain_xml_enabled,omitempty"`
	CustomDomains         []string               `jsonapi:"attr,custom_domains" json:"custom_domains,omitempty"`
	DefaultParams         map[string]interface{} `jsonapi:"attr,default_params" json:"default_params,omitempty"`
	ImageError            string                 `jsonapi:"attr,image_error" json:"image_error,omitempty"`
	ImageErrorAppendQS    bool                   `jsonapi:"attr,image_error_append_qs" json:"image_error_append_qs,omitempty"`
	ImageMissing          string                 `jsonapi:"attr,image_missing" json:"image_missing,omitempty"`
	ImageMissingAppendQS  bool                   `jsonapi:"attr,image_missing_append_qs" json:"image_missing_append_qs,omitempty"`
	ImgixSubdomains       []string               `jsonapi:"attr,imgix_subdomains" json:"imgix_subdomains,omitempty"`
	SecureURLEnabled      bool                   `jsonapi:"attr,secure_url_enabled" json:"secure_url_enabled,omitempty"`
	Type                  string                 `jsonapi:"attr,type" json:"type,omitempty"`

	// AWS S3 Specific Fields
	S3AccessKey string  `jsonapi:"attr,s3_access_key" json:"s3_access_key,omitempty"`
	S3SecretKey string  `jsonapi:"attr,s3_secret_key" json:"s3_secret_key,omitempty"`
	S3Bucket    string  `jsonapi:"attr,s3_bucket" json:"s3_bucket,omitempty"`
	S3Prefix    *string `jsonapi:"attr,s3_prefix" json:"s3_prefix,omitempty"`
}

type ImgixClient struct {
	client       http.Client
	upsertByName bool
}

func NewImgixClient(authToken string, upsertByName bool) *ImgixClient {
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: AuthenticatedRateLimitedTransport{
			roundTripper: http.DefaultTransport,
			rateLimiter:  rate.NewLimiter(rate.Every(2*time.Second), 1), // 1 request(s) every 2 seconds
			token:        authToken,
		},
	}
	return &ImgixClient{client: *client, upsertByName: upsertByName}
}

type AuthenticatedRateLimitedTransport struct {
	roundTripper http.RoundTripper
	rateLimiter  *rate.Limiter
	token        string
}

func (mrt AuthenticatedRateLimitedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Use our RateLimiter
	err := mrt.rateLimiter.Wait(r.Context())
	if err != nil {
		return nil, err
	}
	// Set proper headers
	r.Header.Add("Authorization", "Bearer "+mrt.token)
	r.Header.Add("Accept", jsonapi.MediaType)
	return mrt.roundTripper.RoundTrip(r)
}

func (c *ImgixClient) GetSourceByID(ctx context.Context, resourceId string) (*ImgixSource, error) {
	source := new(ImgixSource)
	if resourceId == "" {
		return source, fmt.Errorf("missing resourceId, can't call GetSourceByID")
	}
	resp, err := c.client.Get(BASE_URL + ImgixResourceSource + "/" + resourceId)
	if err != nil {
		return source, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()

	}
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return source, fmt.Errorf("failed to read body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(reqBody))
	if err := jsonapi.UnmarshalPayload(resp.Body, source); err != nil {
		if resp.StatusCode == 200 {
			return nil, fmt.Errorf("failed to unmarshal jsonapi data: %w", err)
		} else {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(reqBody))
		}
	}
	return source, nil
}

func (c *ImgixClient) GetSourceByName(ctx context.Context, sourceName string) (*ImgixSource, error) {
	if sourceName == "" {
		return nil, fmt.Errorf("missing sourceName, can't call GetSourceByName")
	}
	resp, err := c.client.Get(BASE_URL + ImgixResourceSource + "?filter[name]=" + sourceName)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(reqBody))
	sources, err := jsonapi.UnmarshalManyPayload(resp.Body, reflect.TypeOf(new(ImgixSource)))
	if err != nil {
		if resp.StatusCode == 200 {
			return nil, fmt.Errorf("failed to unmarshal jsonapi data: %w", err)
		} else {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(reqBody))
		}
	}
	if len(sources) == 0 {
		return nil, nil
	} else if len(sources) == 1 {
		return sources[0].(*ImgixSource), nil
	}
	return nil, fmt.Errorf("more than one source was found with name: %s; can't import", sourceName)
}

func (c *ImgixClient) CreateSource(ctx context.Context, source *ImgixSource) (*ImgixSource, error) {
	payload, err := jsonapi.Marshal(source)
	if err != nil {
		return source, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return source, err
	}
	bodyReader := bytes.NewReader(b)
	resp, err := c.client.Post(BASE_URL+ImgixResourceSource, jsonapi.MediaType, bodyReader)
	if err != nil {
		return source, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	remoteSource := new(ImgixSource)
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return source, fmt.Errorf("failed to read body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(reqBody))
	if err := jsonapi.UnmarshalPayload(resp.Body, remoteSource); err != nil {
		if resp.StatusCode == 200 {
			return nil, fmt.Errorf("failed to unmarshal jsonapi data: %w", err)
		} else {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(reqBody))
		}
	}
	return remoteSource, nil
}

func (c *ImgixClient) UpdateSource(ctx context.Context, source *ImgixSource) (*ImgixSource, error) {
	if source.ID == "" {
		return nil, fmt.Errorf("missing ID, can't call UpdateSource")
	}
	payload, err := jsonapi.Marshal(source)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	bodyReader := bytes.NewReader(b)
	req, err := http.NewRequest("PATCH", BASE_URL+ImgixResourceSource+"/"+source.ID, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", jsonapi.MediaType)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	remoteSource := new(ImgixSource)
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return source, fmt.Errorf("failed to read body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(reqBody))
	if err := jsonapi.UnmarshalPayload(resp.Body, remoteSource); err != nil {
		if resp.StatusCode == 200 {
			return nil, fmt.Errorf("failed to unmarshal jsonapi data: %w", err)
		} else {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(reqBody))
		}
	}
	return remoteSource, nil
}

func (c *ImgixClient) DeleteSourceByID(ctx context.Context, resourceId string) error {
	if resourceId == "" {
		return fmt.Errorf("missing resourceId, can't call DeleteSourceByID")
	}
	f := false
	payload, err := jsonapi.Marshal(&ImgixSource{ID: resourceId, Enabled: &f})
	if err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(b)
	req, err := http.NewRequest("PATCH", BASE_URL+ImgixResourceSource+"/"+resourceId, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", jsonapi.MediaType)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(reqBody))
	}
	return nil
}
