package helperfuncs

import (
	"dolos-dev/pkg/structs"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// Config holds configuration for HTTP proxy settings. See
// FromEnvironment for details.
type Config struct {
	// HTTPProxy represents the value of the HTTP_PROXY or
	// http_proxy environment variable. It will be used as the proxy
	// URL for HTTP requests unless overridden by NoProxy.
	HTTPProxy string

	// HTTPSProxy represents the HTTPS_PROXY or https_proxy
	// environment variable. It will be used as the proxy URL for
	// HTTPS requests unless overridden by NoProxy.
	HTTPSProxy string

	// NoProxy represents the NO_PROXY or no_proxy environment
	// variable. It specifies a string that contains comma-separated values
	// specifying hosts that should be excluded from proxying. Each value is
	// represented by an IP address prefix (1.2.3.4), an IP address prefix in
	// CIDR notation (1.2.3.4/8), a domain name, or a special DNS label (*).
	// An IP address prefix and domain name can also include a literal port
	// number (1.2.3.4:80).
	// A domain name matches that name and all subdomains. A domain name with
	// a leading "." matches subdomains only. For example "foo.com" matches
	// "foo.com" and "bar.foo.com"; ".y.com" matches "x.y.com" but not "y.com".
	// A single asterisk (*) indicates that no proxying should be done.
	// A best effort is made to parse the string and errors are
	// ignored.
	NoProxy string

	// CGI holds whether the current process is running
	// as a CGI handler (FromEnvironment infers this from the
	// presence of a REQUEST_METHOD environment variable).
	// When this is set, ProxyForURL will return an error
	// when HTTPProxy applies, because a client could be
	// setting HTTP_PROXY maliciously. See https://golang.org/s/cgihttpproxy.
	CGI bool
}

// proxyConfig holds the parsed configuration for HTTP proxy settings.
type proxyConfig struct {
	// Config represents the original configuration as defined above.
	Config

	// httpsProxy is the parsed URL of the HTTPSProxy if defined.
	httpsProxy *url.URL

	// httpProxy is the parsed URL of the HTTPProxy if defined.
	httpProxy *url.URL

	// ipMatchers represent all values in the NoProxy that are IP address
	// prefixes or an IP address in CIDR notation.
	ipMatchers []matcher

	// domainMatchers represent all values in the NoProxy that are a domain
	// name or hostname & domain name
	domainMatchers []matcher
}

//FindNextProxy finds the next valid proxy in the given list for the given webshop and returns it
func FindNextProxy(amount int, currentProxies []*structs.Proxy, proxies []*structs.Proxy, webshopKind structs.Webshop, proxyLifetime int) ([]*structs.Proxy, error) {

	//reset current proxies
	for _, currentProxy := range currentProxies {
		currentProxy.LastUsedAmazon = time.Now()
		currentProxy.InUse = false
	}

	foundProxies := make([]*structs.Proxy, 0)
	foundCount := 0
	//find next suitable proxies
	for _, proxy := range proxies {
		if !proxy.InUse && getRelevantLastUsedTime(webshopKind, proxy).Add(time.Duration(proxyLifetime)*time.Minute).Before(time.Now()) {
			foundProxies = append(foundProxies, proxy)
			foundCount++
			if foundCount >= amount {
				break
			}
		}
	}

	if len(foundProxies) < amount {
		return nil, fmt.Errorf("Not enough suitable proxies found")
	}

	//mark found proxies as in use
	for _, foundProxy := range foundProxies {
		foundProxy.LastUsedAmazon = time.Now()
		foundProxy.InUse = true
	}

	return foundProxies, nil

}

func getRelevantLastUsedTime(webshopKind structs.Webshop, proxy *structs.Proxy) time.Time {
	switch webshopKind {
	case structs.WEBSHOP_AMAZON, structs.WEBSHOP_AMAZONNL, structs.WEBSHOP_AMAZONDE, structs.WEBSHOP_AMAZONIT, structs.WEBSHOP_AMAZONFR:
		return proxy.LastUsedAmazon
	}

	return time.Now()
}

func FromStr(proxyStr string) *Config {
	return &Config{
		HTTPProxy:  proxyStr,
		HTTPSProxy: proxyStr,
		//NoProxy:    getEnvAny("NO_PROXY", "no_proxy"),
	}
}

func ProxyFunc(cfg *Config) func(reqURL *url.URL) (*url.URL, error) {
	// Preprocess the Config settings for more efficient evaluation.
	cfg1 := &proxyConfig{
		Config: *cfg,
	}
	initProxyConfig(cfg1)
	return proxyForURL(cfg1)
}

func proxyForURL(cfg *proxyConfig) func(reqURL *url.URL) (*url.URL, error) {
	return func(reqURL *url.URL) (*url.URL, error) {
		var proxy *url.URL
		if reqURL.Scheme == "https" {
			proxy = cfg.httpsProxy
		} else if reqURL.Scheme == "http" {
			proxy = cfg.httpProxy
			if proxy != nil && cfg.CGI {
				return nil, errors.New("refusing to use HTTP_PROXY value in CGI environment; see golang.org/s/cgihttpproxy")
			}
		}
		if proxy == nil {
			return nil, nil
		}
		if !useProxy(cfg, canonicalAddr(reqURL)) {
			return nil, nil
		}

		return proxy, nil
	}

}

// useProxy reports whether requests to addr should use a proxy,
// according to the NO_PROXY or no_proxy environment variable.
// addr is always a canonicalAddr with a host and port.
func useProxy(cfg *proxyConfig, addr string) bool {
	if len(addr) == 0 {
		return true
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return false
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() {
			return false
		}
	}

	addr = strings.ToLower(strings.TrimSpace(host))

	if ip != nil {
		for _, m := range cfg.ipMatchers {
			if m.match(addr, port, ip) {
				return false
			}
		}
	}
	for _, m := range cfg.domainMatchers {
		if m.match(addr, port, ip) {
			return false
		}
	}
	return true
}

var portMap = map[string]string{
	"http":   "80",
	"https":  "443",
	"socks5": "1080",
}

// canonicalAddr returns url.Host but always with a ":port" suffix
func canonicalAddr(url *url.URL) string {
	addr := url.Hostname()
	if v, err := idnaASCII(addr); err == nil {
		addr = v
	}
	port := url.Port()
	if port == "" {
		port = portMap[url.Scheme]
	}
	return net.JoinHostPort(addr, port)
}

func idnaASCII(v string) (string, error) {
	// TODO: Consider removing this check after verifying performance is okay.
	// Right now punycode verification, length checks, context checks, and the
	// permissible character tests are all omitted. It also prevents the ToASCII
	// call from salvaging an invalid IDN, when possible. As a result it may be
	// possible to have two IDNs that appear identical to the user where the
	// ASCII-only version causes an error downstream whereas the non-ASCII
	// version does not.
	// Note that for correct ASCII IDNs ToASCII will only do considerably more
	// work, but it will not cause an allocation.
	if isASCII(v) {
		return v, nil
	}
	return idna.Lookup.ToASCII(v)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// matcher represents the matching rule for a given value in the NO_PROXY list
type matcher interface {
	// match returns true if the host and optional port or ip and optional port
	// are allowed
	match(host, port string, ip net.IP) bool
}

// allMatch matches on all possible inputs
type allMatch struct{}

func (a allMatch) match(host, port string, ip net.IP) bool {
	return true
}

type cidrMatch struct {
	cidr *net.IPNet
}

func (m cidrMatch) match(host, port string, ip net.IP) bool {
	return m.cidr.Contains(ip)
}

type ipMatch struct {
	ip   net.IP
	port string
}

func (m ipMatch) match(host, port string, ip net.IP) bool {
	if m.ip.Equal(ip) {
		return m.port == "" || m.port == port
	}
	return false
}

type domainMatch struct {
	host string
	port string

	matchHost bool
}

func (m domainMatch) match(host, port string, ip net.IP) bool {
	if strings.HasSuffix(host, m.host) || (m.matchHost && host == m.host[1:]) {
		return m.port == "" || m.port == port
	}
	return false
}

func initProxyConfig(c *proxyConfig) {
	if parsed, err := parseProxy(c.HTTPProxy); err == nil {
		c.httpProxy = parsed
	}
	if parsed, err := parseProxy(c.HTTPSProxy); err == nil {
		c.httpsProxy = parsed
	}

	for _, p := range strings.Split(c.NoProxy, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if len(p) == 0 {
			continue
		}

		if p == "*" {
			c.ipMatchers = []matcher{allMatch{}}
			c.domainMatchers = []matcher{allMatch{}}
			return
		}

		// IPv4/CIDR, IPv6/CIDR
		if _, pnet, err := net.ParseCIDR(p); err == nil {
			c.ipMatchers = append(c.ipMatchers, cidrMatch{cidr: pnet})
			continue
		}

		// IPv4:port, [IPv6]:port
		phost, pport, err := net.SplitHostPort(p)
		if err == nil {
			if len(phost) == 0 {
				// There is no host part, likely the entry is malformed; ignore.
				continue
			}
			if phost[0] == '[' && phost[len(phost)-1] == ']' {
				phost = phost[1 : len(phost)-1]
			}
		} else {
			phost = p
		}
		// IPv4, IPv6
		if pip := net.ParseIP(phost); pip != nil {
			c.ipMatchers = append(c.ipMatchers, ipMatch{ip: pip, port: pport})
			continue
		}

		if len(phost) == 0 {
			// There is no host part, likely the entry is malformed; ignore.
			continue
		}

		// domain.com or domain.com:80
		// foo.com matches bar.foo.com
		// .domain.com or .domain.com:port
		// *.domain.com or *.domain.com:port
		if strings.HasPrefix(phost, "*.") {
			phost = phost[1:]
		}
		matchHost := false
		if phost[0] != '.' {
			matchHost = true
			phost = "." + phost
		}
		c.domainMatchers = append(c.domainMatchers, domainMatch{host: phost, port: pport, matchHost: matchHost})
	}
}

func parseProxy(proxy string) (*url.URL, error) {
	if proxy == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil ||
		(proxyURL.Scheme != "http" &&
			proxyURL.Scheme != "https" &&
			proxyURL.Scheme != "socks5") {
		// proxy was bogus. Try prepending "http://" to it and
		// see if that parses correctly. If not, we fall
		// through and complain about the original one.
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("invalid proxy address %q: %v", proxy, err)
	}
	return proxyURL, nil
}
