package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
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

// ----------------------------------------------------------------
// Resource: appscan_application
// ----------------------------------------------------------------

func resourceAppScanApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppScanApplicationCreate,
		Read:   resourceAppScanApplicationRead,
		Update: resourceAppScanApplicationUpdate,
		Delete: resourceAppScanApplicationDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the application.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the application.",
			},
			// New required attribute.
			"asset_group_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The asset group ID to which this application belongs.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the application.",
			},
		},
	}
}

func resourceAppScanApplicationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	assetGroupID := d.Get("asset_group_id").(string)
	payload := map[string]interface{}{
		"Name":         d.Get("name").(string),
		"Description":  d.Get("description").(string),
		"AssetGroupId": assetGroupID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v4/Apps", client.ApiEndpoint)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create application, status: %s", resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	id, ok := result["Id"].(string)
	if !ok || id == "" {
		return fmt.Errorf("failed to retrieve application ID from API response")
	}
	d.SetId(id)
	return resourceAppScanApplicationRead(d, m)
}
func resourceAppScanApplicationRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	id := d.Id()

	// Build an OData filter without quotes around the id.
	query := url.Values{}
	query.Set("$filter", fmt.Sprintf("Id eq %s", id))
	urlStr := fmt.Sprintf("%s/api/v4/Apps?%s", client.ApiEndpoint, query.Encode())

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read application, status: %s", resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshal response from the "Items" key instead of "value".
	var result struct {
		Items []map[string]interface{} `json:"Items"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}
	if len(result.Items) == 0 {
		d.SetId("")
		return nil
	}
	app := result.Items[0]
	if v, ok := app["Name"].(string); ok {
		d.Set("name", v)
	}
	if v, ok := app["Description"].(string); ok {
		d.Set("description", v)
	}
	return nil
}

func resourceAppScanApplicationUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	id := d.Id()

	// asset_group_id is ForceNew so it's not updated.
	payload := map[string]interface{}{
		"Name":        d.Get("name").(string),
		"Description": d.Get("description").(string),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v4/Apps/%s", client.ApiEndpoint, id)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update application, status: %s", resp.Status)
	}
	return resourceAppScanApplicationRead(d, m)
}

func resourceAppScanApplicationDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	id := d.Id()

	url := fmt.Sprintf("%s/api/v4/Apps/%s", client.ApiEndpoint, id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete application, status: %s", resp.Status)
	}
	d.SetId("")
	return nil
}

// ----------------------------------------------------------------
// Data Source: appscan_asset_groups (list)
// ----------------------------------------------------------------

func dataSourceAssetGroups() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAssetGroupsRead,
		Schema: map[string]*schema.Schema{
			// Optional "name" argument to filter the list.
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "If provided, only asset groups with this exact name are returned.",
			},
			"asset_groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of asset groups.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the asset group.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the asset group.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the asset group.",
						},
					},
				},
			},
		},
	}
}

func dataSourceAssetGroupsRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)

	// Build the OData filter if a "name" is provided.
	var filterQuery string
	if name, ok := d.GetOk("name"); ok {
		filterQuery = fmt.Sprintf("Name eq '%s'", name.(string))
	}
	query := url.Values{}
	if filterQuery != "" {
		query.Set("$filter", filterQuery)
	}

	urlStr := fmt.Sprintf("%s/api/v4/AssetGroups?%s", client.ApiEndpoint, query.Encode())
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read asset groups, status: %s", resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Items []struct {
			Id          string `json:"Id"`
			Name        string `json:"Name"`
			Description string `json:"Description"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	groups := make([]interface{}, len(result.Items))
	for i, ag := range result.Items {
		group := map[string]interface{}{
			"id":          ag.Id,
			"name":        ag.Name,
			"description": ag.Description,
		}
		groups[i] = group
	}

	if err := d.Set("asset_groups", groups); err != nil {
		return err
	}
	d.SetId("asset_groups")
	return nil
}

// ----------------------------------------------------------------
// Data Source: appscan_asset_group (single asset group by name)
// ----------------------------------------------------------------

func dataSourceAssetGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAssetGroupRead,
		Schema: map[string]*schema.Schema{
			// The asset group name is required to uniquely identify one.
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the asset group to retrieve.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the asset group.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the asset group.",
			},
		},
	}
}

func dataSourceAssetGroupRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	assetName := d.Get("name").(string)

	// Build OData filter from the provided name.
	filterQuery := fmt.Sprintf("Name eq '%s'", assetName)
	query := url.Values{}
	query.Set("$filter", filterQuery)

	urlStr := fmt.Sprintf("%s/api/v4/AssetGroups?%s", client.ApiEndpoint, query.Encode())
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.ApiToken))

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read asset group, status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Items []struct {
			Id          string `json:"Id"`
			Name        string `json:"Name"`
			Description string `json:"Description"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Items) == 0 {
		return fmt.Errorf("no asset group found with name: %s", assetName)
	}
	if len(result.Items) > 1 {
		return fmt.Errorf("multiple asset groups found with name: %s", assetName)
	}

	asset := result.Items[0]
	d.SetId(asset.Id)
	if err := d.Set("name", asset.Name); err != nil {
		return err
	}
	if err := d.Set("description", asset.Description); err != nil {
		return err
	}
	return nil
}

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return Provider()
		},
	})
}
