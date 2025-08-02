package subnet

import (
	"fmt"
	"io"
	"net/http"
)

func getPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org") // returns plain text IP
	if err != nil {
		return "", fmt.Errorf("query external server: %w", err)
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read external server response: %w", err)
	}
	return string(ip), nil
}

func SetHeader() (func(*http.Request), error) {
	publicIP, err := getPublicIP()
	if err != nil {
		return nil, fmt.Errorf("get public IP: %w", err)
	}
	return func(r *http.Request) {
		r.Header.Set("X-Real-IP", publicIP)
	}, nil
}
