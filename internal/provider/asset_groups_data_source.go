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
