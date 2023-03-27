package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/jsonapi"
)

const BASE_URL = "https://api.imgix.com/"

const (
	ImgixResourceSource string = "sources"
	ImgixResourcePurge  string = "purges"
	ImgixResourceReport string = "reports"
)

type ImgixSource struct {
	ID               string                `jsonapi:"primary,sources"`
	Name             string                `jsonapi:"attr,name"`
	Enabled          bool                  `jsonapi:"attr,enabled"`
	Deployment       ImgixSourceDeployment `jsonapi:"attr,deployment"`
	DeploymentStatus string                `jsonapi:"attr,deployment_status"`
	SecureURLToken   string                `jsonapi:"attr,secure_url_token"`
	DateDeployed     int                   `jsonapi:"attr,date_deployed"`
}

type ImgixSourceDeployment struct {
	AllowsUpload          bool                   `jsonapi:"attr,allows_upload"`
	Annotation            string                 `jsonapi:"attr,annotation"`
	CacheTTLBehavior      string                 `jsonapi:"attr,cache_ttl_behavior"`
	CacheTTLError         int                    `jsonapi:"attr,cache_ttl_error"`
	CacheTTLValue         int                    `jsonapi:"attr,cache_ttl_value"`
	CrossdomainXMLEnabled bool                   `jsonapi:"attr,crossdomain_xml_enabled"`
	CustomDomains         []string               `jsonapi:"attr,custom_domains"`
	DefaultParams         map[string]interface{} `jsonapi:"attr,default_params"`
	ImageError            string                 `jsonapi:"attr,image_error"`
	ImageErrorAppendQS    bool                   `jsonapi:"attr,image_error_append_qs"`
	ImageMissing          string                 `jsonapi:"attr,image_missing"`
	ImageMissingAppendQS  bool                   `jsonapi:"attr,image_missing_append_qs"`
	ImgixSubdomains       []string               `jsonapi:"attr,imgix_subdomains"`
	SecureURLEnabled      bool                   `jsonapi:"attr,secure_url_enabled"`
	Type                  string                 `jsonapi:"attr,type"`

	// AWS S3 Specific Fields
	S3AccessKey string `jsonapi:"attr,s3_access_key"`
	S3SecretKey string `jsonapi:"attr,s3_secret_key"`
	S3Bucket    string `jsonapi:"attr,s3_bucket"`
	S3Prefix    string `jsonapi:"attr,s3_prefix"`
}

type Client interface {
	SetAuthToken(authToken string)
	GetSourceByID(resourceId string) (*ImgixSource, error)
	CreateSource(source ImgixSource) (*ImgixSource, error)
}

type ImgixClient struct {
	client http.Client
}

type AuthenticatedTransport struct {
	r http.RoundTripper
	t string
}

func (mrt AuthenticatedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Authorization", "Bearer "+mrt.t)
	r.Header.Add("Accept", jsonapi.MediaType)
	r.Header.Add("Content-Type", jsonapi.MediaType)
	return mrt.r.RoundTrip(r)
}

func (c *ImgixClient) SetAuthToken(authToken string) {
	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: AuthenticatedTransport{r: http.DefaultTransport, t: authToken},
	}
	c.client = *client
}

func (c *ImgixClient) GetSourceByID(resourceId string) (*ImgixSource, error) {
	source := new(ImgixSource)
	resp, err := c.client.Get(BASE_URL + "/api/v1/" + ImgixResourceSource + "/" + resourceId)
	if err != nil {
		return source, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if err := jsonapi.UnmarshalPayload(resp.Body, source); err != nil {
		return source, err
	}
	return source, nil
}

func (c *ImgixClient) CreateSource(source *ImgixSource) (*ImgixSource, error) {
	payload, err := jsonapi.Marshal(source)
	if err != nil {
		return source, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return source, err
	}
	bodyReader := bytes.NewReader(b)
	resp, err := c.client.Post(BASE_URL+"/api/v1/"+ImgixResourceSource+"/", jsonapi.MediaType, bodyReader)
	if err != nil {
		return source, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	remoteSource := new(ImgixSource)
	if err := jsonapi.UnmarshalPayload(resp.Body, remoteSource); err != nil {
		return remoteSource, err
	}
	return remoteSource, nil
}
