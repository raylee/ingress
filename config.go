package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/inetaf/tcpproxy"
)

type Rule struct {
	protoPort, match, dest string
}

type Proxy struct {
	config string
	routes map[string][]Rule
	*tcpproxy.Proxy
}

func (c *Proxy) String() (s string) {
	keys := []string{}
	for k := range c.routes {
		keys = append(keys, k)
	}
	parsePort := func(s string) (port int) {
		p := strings.Index(s, ":")
		port, _ = strconv.Atoi(s[p+1:])
		return
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, pj := parsePort(keys[i]), parsePort(keys[j])
		return pi < pj
	})

	for _, listen := range keys {
		s += fmt.Sprintf("%s\n", listen)
		rules := c.routes[listen]
		width := 0
		for i := range rules {
			w := len(rules[i].match)
			if w > width {
				width = w
			}
		}
		for _, r := range c.routes[listen] {
			s += fmt.Sprintf("\t%-12s %*s → %s\n", r.protoPort, width, r.match, r.dest)
		}
	}
	return
}

// addPort takes a base and adds a colon and port number if it's missing.
func addPort(base string, port int) string {
	if strings.Index(base, ":") > -1 {
		return base
	}
	return fmt.Sprintf("%s:%d", base, port)
}

// tokens removes any comments and returns the whitespace-separated tokens on line.
func tokens(line string) []string {
	// remove any comments to end of line
	if ci := strings.Index(line, "#"); ci >= 0 {
		line = line[:ci]
	}
	if ci := strings.Index(line, "//"); ci >= 0 {
		line = line[:ci]
	}
	return strings.Fields(line)
}

func NewProxy(conf string) (*Proxy, error) {
	c := &Proxy{config: conf, Proxy: &tcpproxy.Proxy{}, routes: make(map[string][]Rule)}
	bindAddr := ""
	lines := strings.Split(conf, "\n")

	for _, r := range lines {
		tok := tokens(r)
		if len(tok) == 0 {
			continue
		}
		pp := strings.SplitN(tok[0], "/", 2)
		proto, listenPort := pp[0], ""
		if len(pp) == 2 {
			listenPort = pp[1]
		}

		switch {
		case tok[0] == "bind.address" && len(tok) == 2:
			bindAddr = tok[1]

		case proto == "tcp" && len(tok) == 2:
			// tcp/9000 tiassa:3000
			dest := tok[1]
			from := bindAddr + ":" + listenPort
			c.Proxy.AddRoute(from, tcpproxy.To(dest))
			c.routes[from] = append(c.routes[from], Rule{"tcp/" + listenPort, "", dest})

		case proto == "http" && len(tok) == 3:
			// http[/80] icbm.evq.io 127.0.0.2
			match, dest := tok[1], addPort(tok[2], 80)
			if listenPort == "" {
				listenPort = "80"
			}
			from := bindAddr + ":" + listenPort
			c.Proxy.AddHTTPHostRoute(from, match, tcpproxy.To(dest))
			c.routes[from] = append(c.routes[from], Rule{"http/" + listenPort, match, dest})

		case proto == "https" && len(tok) == 3:
			// https[/443] tandem.evq.io 127.0.0.3
			match, dest := tok[1], addPort(tok[2], 443)
			if listenPort == "" {
				listenPort = "443"
			}
			from := bindAddr + ":" + listenPort
			c.Proxy.AddSNIRoute(from, match, tcpproxy.To(dest))
			c.routes[from] = append(c.routes[from], Rule{"https/" + listenPort, match, dest})

		default:
			return nil, fmt.Errorf("couldn't parse route declaration: %s", r)
		}
	}

	return c, nil
}
