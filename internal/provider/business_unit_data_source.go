package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBusinessUnit() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceBusinessUnitRead,
		Schema: map[string]*schema.Schema{
			// The BusinessUnit name is required to uniquely identify one.
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the BusinessUnit to retrieve.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the BusinessUnit.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the BusinessUnit.",
			},
		},
	}
}

func dataSourceBusinessUnitRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*AppScanClient)
	buName := d.Get("name").(string)

	// Build the OData filter using the provided name.
	filterQuery := fmt.Sprintf("Name eq '%s'", buName)
	query := url.Values{}
	query.Set("$filter", filterQuery)

	// Call the API GET /api/v4/BusinessUnits with the filter.
	urlStr := fmt.Sprintf("%s/api/v4/BusinessUnits?%s", client.ApiEndpoint, query.Encode())
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
		return fmt.Errorf("failed to read BusinessUnit, status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// The expected result contains an array of items.
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
		return fmt.Errorf("no BusinessUnit found with name: %s", buName)
	}
	if len(result.Items) > 1 {
		return fmt.Errorf("multiple BusinessUnits found with name: %s", buName)
	}

	bu := result.Items[0]
	d.SetId(bu.Id)
	if err := d.Set("name", bu.Name); err != nil {
		return err
	}
	if err := d.Set("description", bu.Description); err != nil {
		return err
	}
	return nil
}
