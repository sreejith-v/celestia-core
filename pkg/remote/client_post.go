package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *Client) SendBatch(ctx context.Context, evs []Event) error {
	br := BatchRequest{
		NodeID:  c.nodeID,
		ChainID: c.chainID,
		Events:  evs,
	}

	rbr, err := json.Marshal(br)
	if err != nil {
		fmt.Println("marshal error: ", err)
		return err
	}

	resp, err := c.sendRequest(http.MethodPost, c.CreateServerURL(batchMethod, ""), rbr)
	if err != nil {
		fmt.Println("sedn request error: ", err)
		return err
	}

	var gr GenericResponse
	err = json.Unmarshal(resp, &gr)
	if err != nil {
		fmt.Println("1 unmarshal error: ", err, string(resp))
		return err
	}

	if gr.Status != EventResponseStatusOK {
		return fmt.Errorf("failure to send batch to server: %s", gr.Error)
	}
	return nil
}

func (c *Client) sendRequest(method, url string, requestBody []byte) ([]byte, error) {
	// Create a new HTTP request.
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	// Set the request headers.
	req.Header.Set("Content-Type", "application/json")

	// Send the request and get the response.
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body.
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}
