package subnet

import (
	"fmt"
	"net"
	"net/http"
)

type verifier struct {
	ipNet        *net.IPNet
	shouldVerify func(req *http.Request) bool
}

func (v *verifier) verifyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if v.shouldVerify(req) {
			ipStr := req.Header.Get("X-Real-IP")
			ip := net.ParseIP(ipStr)
			if ip == nil {
				http.Error(w, fmt.Sprintf("invalid IP: %s", ipStr),
					http.StatusBadRequest)
				return
			}
			if !v.ipNet.Contains(ip) {
				http.Error(w, "read failed", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func VerifyHeader(
	trustedSubnet string, shouldVerify func(req *http.Request) bool,
) (func(http.Handler) http.Handler, error) {
	_, ipNet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %s", trustedSubnet)
	}

	v := &verifier{
		ipNet:        ipNet,
		shouldVerify: shouldVerify,
	}
	return v.verifyHeaderMiddleware, nil
}
