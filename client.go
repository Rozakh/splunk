package splunk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

var (
	client       *http.Client
	transport    *http.Transport
	jsonResponse map[string]string
)

//Client for the Splunk REST API.
type Client struct {
	host, sessionKey string
}

type searchResults struct {
	Results []map[string]string
}

func init() {
	transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client = &http.Client{Transport: transport}
}

//New Splunk client for the provided host.
func New(host string) *Client {
	return &Client{host: host}
}

//Login to Splunk with provided credentials.
func (c *Client) Login(username, password string) error {
	data := make(url.Values)
	data.Add("username", username)
	data.Add("password", password)
	data.Add("output_mode", "json")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/services/auth/login", c.host),
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(body, &jsonResponse)
	c.sessionKey = jsonResponse["sessionKey"]
	return nil
}

//Search and return results as key:value maps, including fields, extracted by regex groups.
func (c *Client) Search(searchString string, resultNumber int) ([]map[string]string, error) {
	data := make(url.Values)
	searchQuery := fmt.Sprintf(`search index=* | regex "%s" | head %s | rex "%s"`, searchString,
		strconv.Itoa(resultNumber), searchString)
	data.Add("search", searchQuery)
	data.Add("exec_mode", "blocking")
	data.Add("output_mode", "json")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/services/search/jobs", c.host),
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", c.sessionKey))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(body, &jsonResponse)
	searchID := jsonResponse["sid"]
	result, err := c.getResult(searchID)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) getResult(searchID string) ([]map[string]string, error) {
	var result searchResults
	data := make(url.Values)
	data.Add("output_mode", "json")
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/services/search/jobs/%s/results", c.host, searchID),
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", c.sessionKey))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(body, &result)
	return result.Results, nil
}
