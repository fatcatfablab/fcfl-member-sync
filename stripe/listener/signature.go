package listener

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func parseHeader(header string) (map[string]string, error) {
	elemMap := make(map[string]string)
	elemSlice := strings.Split(header, ",")
	for _, e := range elemSlice {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("malformed signature header: %q", header)
		}
		elemMap[kv[0]] = kv[1]
	}

	return elemMap, nil
}

func sign(payload []byte, timestamp string, secret string) ([]byte, error) {
	signedPayload := fmt.Appendf(nil, "%s.%s", timestamp, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(signedPayload)
	if err != nil {
		return nil, fmt.Errorf("error writing to hmac: %v", err)
	}
	return mac.Sum(nil), nil
}

func verifySignature(payload []byte, signature string, secret string) error {
	elemMap, err := parseHeader(signature)
	if err != nil {
		return err
	}

	expectedMAC, err := sign(payload, elemMap["t"], secret)
	if err != nil {
		return err
	}

	v1Header := []byte(elemMap["v1"])
	receivedMAC := make([]byte, hex.DecodedLen(len(v1Header)))
	_, err = hex.Decode(receivedMAC, v1Header)
	if err != nil {
		return fmt.Errorf("error decoding header: %v", err)
	}

	if !hmac.Equal(expectedMAC, receivedMAC) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
