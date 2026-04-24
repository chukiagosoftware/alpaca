package google_places

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
)

// Not required for Places API only for Maps
func SignURL(inputURL, secret string) (string, error) {
	if inputURL == "" || secret == "" {
		return "", fmt.Errorf("both inputURL and secret are required")
	}

	u, err := url.Parse(inputURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Strip protocol and host, keep path + query
	urlToSign := u.Path + "?" + u.RawQuery

	// Decode the URL signing secret from modified Base64
	decodedKey, err := base64.URLEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to decode secret: %w", err)
	}

	// Sign with HMAC-SHA1
	h := hmac.New(sha1.New, decodedKey)
	h.Write([]byte(urlToSign))
	signature := h.Sum(nil)

	// Encode signature to modified Base64
	encodedSignature := base64.URLEncoding.EncodeToString(signature)

	// Append signature to original URL
	signedURL := inputURL + "&signature=" + encodedSignature
	return signedURL, nil
}
