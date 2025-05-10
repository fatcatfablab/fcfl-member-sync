package listener

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func verifySignature(payload []byte, signature string, secret string) error {
	elemMap := make(map[string]string)
	elemSlice := strings.Split(signature, ",")
	for _, e := range elemSlice {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("malformed signature header: %q", signature)
		}
		elemMap[kv[0]] = kv[1]
	}

	signedPayload := fmt.Appendf(nil, "%s.%s", elemMap["t"], payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(signedPayload)
	if err != nil {
		return fmt.Errorf("error writing to hmac: %v", err)
	}
	expectedMAC := mac.Sum(nil)

	v1Header := []byte(elemMap["v1"])
	actualMAC := make([]byte, hex.DecodedLen(len(v1Header)))
	_, err = hex.Decode(actualMAC, v1Header)
	if err != nil {
		return fmt.Errorf("error decoding header: %v", err)
	}

	if !hmac.Equal(expectedMAC, actualMAC) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
