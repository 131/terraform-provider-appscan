package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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
