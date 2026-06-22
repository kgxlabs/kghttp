package kghttp

type RoundTripper interface {
	RoundTrip(req *Request) (*Response, error)
}
