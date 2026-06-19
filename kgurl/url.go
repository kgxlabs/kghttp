package kgurl

import "net/url"

type URL struct {
	*url.URL
}

func Parse(rawURL string) (*URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &URL{URL: u}, nil
}
