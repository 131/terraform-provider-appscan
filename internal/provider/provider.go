package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// AppScanClient holds configuration for API communication.
type AppScanClient struct {
	ApiEndpoint string
	ApiToken    string
	Client      *http.Client
}

// providerConfigure authenticates via /api/v4/Account/ApiKeyLogin using key_id and key_secret.
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	endpoint := d.Get("api_endpoint").(string)
	keyID := d.Get("key_id").(string)
	keySecret := d.Get("key_secret").(string)

	// Construct payload for API key login.
	payload := map[string]string{
		"KeyId":     keyID,
		"KeySecret": keySecret,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	loginURL := fmt.Sprintf("%s/api/v4/Account/ApiKeyLogin", endpoint)
	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate via API key, status: %s", resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// The login endpoint now returns a "Token" field.
	var authResp struct {
		Token string `json:"Token"`
	}
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return nil, err
	}
	if authResp.Token == "" {
		return nil, fmt.Errorf("failed to obtain token from API key login response")
	}

	return &AppScanClient{
		ApiEndpoint: endpoint,
		ApiToken:    authResp.Token,
		Client:      client,
	}, nil
}

// Provider returns the Terraform provider for AppScan.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("APPSCAN_API_ENDPOINT", "https://cloud.appscan.com/"),
				Description: "The API endpoint for the AppScan REST API.",
			},
			"key_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("APPSCAN_KEY_ID", nil),
				Description: "The API Key ID for authentication.",
			},
			"key_secret": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("APPSCAN_KEY_SECRET", nil),
				Description: "The API Key Secret for authentication.",
				Sensitive:   true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"appscan_application": resourceAppScanApplication(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"appscan_asset_groups": dataSourceAssetGroups(),
			"appscan_asset_group":  dataSourceAssetGroup(),
		},
		ConfigureFunc: providerConfigure,
	}
}
