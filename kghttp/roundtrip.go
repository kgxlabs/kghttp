package kghttp

type RoundTripper struct {
}

func (rt *RoundTripper) RoundTripper(req *Request) (*Response, error) {
	return nil, nil
}
