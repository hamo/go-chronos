package chronos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

const (
	DEBUG_LEVEL = 10
)

type Client struct {
	sync.RWMutex
	// the configuration for the client
	config *Config
	// the ip address of the client
	ipAddress string
	// the http client use for making requests
	httpClient *http.Client
	// the marathon cluster
	cluster *Cluster
}

func NewClient(config *Config) (*Client, error) {
	cluster, err := newCluster(config.URL)
	if err != nil {
		return nil, err
	}

	re := new(Client)
	re.config = config
	re.cluster = cluster
	re.httpClient = &http.Client{
		Timeout: time.Duration(config.RequestTimeout) * time.Second,
	}
	return re, nil
}

func (c *Client) encodeRequest(data interface{}) (string, error) {
	response, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(response), err
}

func (c *Client) decodeRequest(stream io.Reader, result interface{}) error {
	if err := json.NewDecoder(stream).Decode(result); err != nil {
		return err
	}

	return nil
}

func (c *Client) buildPostData(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}
	content, err := c.encodeRequest(data)
	if err != nil {
		return "", err
	}

	return content, nil
}

func (c *Client) apiGet(uri string, post, result interface{}) error {
	return c.apiOperation("GET", uri, post, result)
}

func (c *Client) apiPut(uri string, post, result interface{}) error {
	return c.apiOperation("PUT", uri, post, result)
}

func (c *Client) apiPost(uri string, post, result interface{}) error {
	return c.apiOperation("POST", uri, post, result)
}

func (c *Client) apiDelete(uri string, post, result interface{}) error {
	return c.apiOperation("DELETE", uri, post, result)
}

func (c *Client) apiOperation(method, uri string, post, result interface{}) error {
	content, err := c.buildPostData(post)
	if err != nil {
		return err
	}

	_, _, err = c.apiCall(method, uri, content, result)

	return err
}

func (c *Client) apiCall(method, uri, body string, result interface{}) (int, string, error) {
	glog.V(DEBUG_LEVEL).Infof("[api]: method: %s, uri: %s, body: %s", method, uri, body)

	status, content, _, err := c.httpRequest(method, uri, body)
	if err != nil {
		return 0, "", err
	}

	glog.V(DEBUG_LEVEL).Infof("[api] result: status: %d, content: %s\n", status, content)
	if status-200 < 100 {
		if result != nil {
			if err := c.decodeRequest(strings.NewReader(content), result); err != nil {
				glog.V(DEBUG_LEVEL).Infof("failed to unmarshall the response from chronos, error: %s", err)
				return status, content, errors.New("invalid response from chronos")
			}
		}
		return status, content, nil
	}

	if content == "" {
		return status, "", fmt.Errorf("API call returns status %d", status)
	} else {
		return status, "", errors.New(content)
	}
}

func (c *Client) httpRequest(method, uri, body string) (int, string, *http.Response, error) {
	var content string

	// step: get a member from the cluster
	chronos, err := c.cluster.GetMember()
	if err != nil {
		return 0, "", nil, err
	}

	url := fmt.Sprintf("%s%s", chronos, uri)

	glog.V(DEBUG_LEVEL).Infof("[http] request: %s, uri: %s, url: %s", method, uri, url)
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return 0, "", nil, err
	}

	// step: add any basic auth and the content headers
	if c.config.HttpBasicAuthUser != "" {
		request.SetBasicAuth(c.config.HttpBasicAuthUser, c.config.HttpBasicPassword)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		c.cluster.MarkInactive()
		// step: retry the request with another endpoint
		return c.httpRequest(method, uri, body)
	}

	if response.ContentLength != 0 {
		responseContent, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return response.StatusCode, "", response, err
		}
		content = string(responseContent)
	}

	return response.StatusCode, content, response, nil
}
