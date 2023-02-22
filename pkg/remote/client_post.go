package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// QueryEvents queries the server for events using the given path. The format of
// the path should be "chainID/nodeID/type". The server will return all events
// that match that query.
func (c *Client) QueryEvents(path string) ([]Event, error) {
	q, err := NewQuery(path)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendRequest(http.MethodGet, c.CreateServerURL(getEventsMethod, q), nil)
	if err != nil {
		return nil, err
	}

	var qr QueryResponse
	err = json.Unmarshal(resp, &qr)
	if err != nil {
		return nil, err
	}

	if qr.Status != EventResponseStatusOK {
		return nil, fmt.Errorf("failure to get file from server: %s", qr.Error)
	}

	if len(qr.Events) == 0 {
		return nil, fmt.Errorf("no events found for query: %s", path)
	}

	return qr.Events, nil
}

// SendBatch sends a batch of events to the server.
func (c *Client) SendBatch(evs []Event) error {
	br := BatchRequest{
		NodeID:  c.nodeID,
		ChainID: c.chainID,
		Events:  evs,
	}

	rbr, err := json.Marshal(br)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(http.MethodPost, c.CreateServerURL(batchMethod, Query{}), rbr)
	if err != nil {
		return err
	}

	var gr GenericResponse
	err = json.Unmarshal(resp, &gr)
	if err != nil {
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
