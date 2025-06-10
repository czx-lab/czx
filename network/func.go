package network

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strings"
)

// https://www.cnblogs.com/flydean/p/16356050.html
// GetClientIPFromProxyProtocol retrieves the client's IP address and port from the Proxy Protocol header if present.
// If the header is not present, it falls back to the remote address of the connection.
// nginx configuration example:
// ```
//
//	stream {
//	   server {
//	       listen 8080 proxy_protocol;
//	   }
//	}
//
//	stream {
//	   server {
//	       listen 8080;
//	       proxy_pass example.com:8081;
//	       proxy_protocol on;
//	   }
//	}
//
//	http {
//	   proxy_set_header X-Real-IP       $proxy_protocol_addr;
//	   proxy_set_header X-Forwarded-For $proxy_protocol_addr;
//	}
//
// ```
func GetClientIPFromProxyProtocol(conn net.Conn) (ip, port *string, err error) {
	ip = new(string)
	port = new(string)
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	if strings.HasPrefix(line, "PROXY") {
		parts := strings.Split(line, " ")
		if len(parts) >= 5 {
			*ip = strings.TrimSpace(parts[2])   // Client IP
			*port = strings.TrimSpace(parts[4]) // Client port
			return
		}

		return nil, nil, errors.New("invalid proxy protocol header")
	}

	// If no Proxy Protocol header, fallback to remote address
	host, rport, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		*ip = conn.RemoteAddr().String()
		return
	}

	*ip = host
	*port = rport
	return
}

// GetClientIP retrieves the client's IP address and port from the HTTP request.
// It checks the "X-Forward-For" header first, then "X-Real-IP", and finally falls back to the remote address.
// nginx configuration example:
// ```
//
//	location / {
//		proxy_pass http://backend;
//		proxy_set_header X-Forward-For $proxy_add_x_forwarded_for;
//		proxy_set_header X-Real-IP $remote_addr;
//	}
//
// ```
func GetClientIP(r *http.Request) (ip, port *string) {
	ip = new(string)
	port = new(string)
	xff := r.Header.Get("X-Forwarded-For")
	if len(xff) > 0 {
		for ipitem := range strings.SplitSeq(xff, ",") {
			ipitem = strings.TrimSpace(ipitem)
			if net.ParseIP(ipitem) != nil {
				*ip = ipitem
				return
			}
		}
	}

	xri := r.Header.Get("X-Real-IP")
	if net.ParseIP(xri) != nil {
		*ip = xri
		return
	}

	rip, rport, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		*ip = r.RemoteAddr
		return
	}

	if net.ParseIP(rip) != nil {
		*ip = rip
		*port = rport
		return
	}

	return
}
