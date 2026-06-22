package kghttp

import "io"

type Client struct {
	Transport RoundTripper
}

var DefaultClient = &Client{
	Transport: DefaultTransport,
}

func (c *Client) Do(req *Request) (*Response, error) {
	return c.Transport.RoundTrip(req)
}

func Get(url string) (*Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return DefaultClient.Do(req)
}

func Head(url string) (*Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	req.Headers.Set("content-length", "0")

	return DefaultClient.Do(req)
}

func Post(url string, contentType string, body io.Reader) (*Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Headers.Set("content-type", contentType)

	return DefaultClient.Do(req)
}
