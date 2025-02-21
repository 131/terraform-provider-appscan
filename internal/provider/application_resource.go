package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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
