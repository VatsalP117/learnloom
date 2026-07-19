package httpapp

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

type HostKind string

const (
	HostApex     HostKind = "apex"
	HostWWW      HostKind = "www"
	HostApp      HostKind = "app"
	HostSite     HostKind = "site"
	HostRejected HostKind = "rejected"
)

type RequestHost struct {
	Kind     HostKind
	Hostname string
	Username string
}

var siteLabelPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,29}$`)
var reservedSiteLabels = map[string]struct{}{
	"admin": {}, "api": {}, "app": {}, "assets": {}, "auth": {},
	"blog": {}, "clerk": {}, "dashboard": {}, "docs": {}, "help": {},
	"learnloom": {}, "mail": {}, "root": {}, "status": {}, "support": {},
	"www": {},
}

func ClassifyHost(value, rootDomain string) (RequestHost, error) {
	hostname, err := normalizeHostHeader(value)
	if err != nil {
		return RequestHost{Kind: HostRejected}, err
	}
	rootDomain = strings.ToLower(strings.TrimSpace(rootDomain))
	switch hostname {
	case rootDomain:
		return RequestHost{Kind: HostApex, Hostname: hostname}, nil
	case "www." + rootDomain:
		return RequestHost{Kind: HostWWW, Hostname: hostname}, nil
	case "app." + rootDomain:
		return RequestHost{Kind: HostApp, Hostname: hostname}, nil
	}
	suffix := "." + rootDomain
	if !strings.HasSuffix(hostname, suffix) {
		return RequestHost{Kind: HostRejected, Hostname: hostname}, errors.New("request Host is not allowed")
	}
	username := strings.TrimSuffix(hostname, suffix)
	if strings.Contains(username, ".") || !siteLabelPattern.MatchString(username) ||
		strings.HasSuffix(username, "-") || strings.Contains(username, "--") {
		return RequestHost{Kind: HostRejected, Hostname: hostname}, errors.New("request Host is not a valid Personal Site")
	}
	if _, reserved := reservedSiteLabels[username]; reserved {
		return RequestHost{Kind: HostRejected, Hostname: hostname}, errors.New("request Host is reserved")
	}
	return RequestHost{Kind: HostSite, Hostname: hostname, Username: username}, nil
}

func normalizeHostHeader(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || strings.ContainsAny(value, `/\@,`) ||
		strings.HasSuffix(value, ".") {
		return "", errors.New("request Host is malformed")
	}
	if strings.HasPrefix(value, "[") {
		host, _, err := net.SplitHostPort(value)
		if err != nil {
			return "", errors.New("request Host is malformed")
		}
		return strings.Trim(host, "[]"), nil
	}
	if strings.Count(value, ":") == 1 {
		host, port, err := net.SplitHostPort(value)
		if err != nil || host == "" || port == "" {
			return "", errors.New("request Host is malformed")
		}
		return host, nil
	}
	if strings.Contains(value, ":") {
		return "", errors.New("request Host is malformed")
	}
	return value, nil
}
