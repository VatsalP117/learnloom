package source

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const userAgent = "learnloom/1.0 (+https://learnloom.blog)"

type fetchResult struct {
	Body         []byte
	ContentType  string
	FinalURL     *url.URL
	ETag         string
	LastModified string
	StatusCode   int
}

func secureTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	resolver := net.DefaultResolver
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve source host: %w", err)
			}
			if len(addresses) == 0 {
				return nil, errors.New("source hostname did not resolve")
			}
			for _, address := range addresses {
				if !isPublicAddress(address) {
					return nil, errors.New("source URL resolves to a non-public address")
				}
			}
			selected := addresses[0].Unmap()
			return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
		},
	}
}

func redirectPolicy(maximum int) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if len(via) > maximum {
			return errors.New("source redirected too many times")
		}
		_, err := validateWebURL(request.URL.String())
		return err
	}
}

func doHTTP(ctx context.Context, client *http.Client, rawURL string, maxBytes int64, accept string, conditionalETag, conditionalModified string) (fetchResult, error) {
	parsed, err := validateWebURL(rawURL)
	if err != nil {
		return fetchResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fetchResult{}, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", userAgent)
	if conditionalETag != "" {
		request.Header.Set("If-None-Match", conditionalETag)
	}
	if conditionalModified != "" {
		request.Header.Set("If-Modified-Since", conditionalModified)
	}
	response, err := client.Do(request)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			return fetchResult{}, fmt.Errorf(
				"source request failed during %s: %w",
				requestErr.Op,
				requestErr.Err,
			)
		}
		return fetchResult{}, errors.New("source request failed")
	}
	defer response.Body.Close()

	result := fetchResult{
		ContentType:  response.Header.Get("Content-Type"),
		FinalURL:     response.Request.URL,
		ETag:         response.Header.Get("ETag"),
		LastModified: response.Header.Get("Last-Modified"),
		StatusCode:   response.StatusCode,
	}

	if result.StatusCode == http.StatusNotModified {
		return result, nil
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return fetchResult{}, fmt.Errorf("source returned HTTP %d", result.StatusCode)
	}
	body, err := readBounded(response.Body, maxBytes)
	if err != nil {
		return fetchResult{}, err
	}
	result.Body = body
	return result, nil
}

func validateWebURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("source URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("source URL must use HTTP or HTTPS")
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return nil, errors.New("source URL must have a host and no credentials")
	}
	if strings.EqualFold(parsed.Hostname(), "localhost") ||
		strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".localhost") {
		return nil, errors.New("source URL resolves to a non-public address")
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && !isPublicAddress(address) {
		return nil, errors.New("source URL resolves to a non-public address")
	}
	return parsed, nil
}

var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
	"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
	"2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20",
)

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() ||
		address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() ||
		address.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func mustPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

func readBounded(reader io.Reader, maximum int64) ([]byte, error) {
	if maximum < 1 {
		return nil, errors.New("response size limit is invalid")
	}
	body, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maximum {
		return nil, errors.New("source exceeded size limit")
	}
	return body, nil
}

func isFeedContentType(value string) bool {
	mediaType, _, _ := mime.ParseMediaType(value)
	switch mediaType {
	case "application/atom+xml", "application/rss+xml", "application/xml",
		"text/xml", "application/feed+json", "application/json",
		"text/plain", "":
		return true
	default:
		return false
	}
}
