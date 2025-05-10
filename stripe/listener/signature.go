package listener

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

func verifySignature(payload []byte, signature string, secret string) bool {
	elemMap := make(map[string]string)
	elemSlice := strings.Split(signature, ",")
	for _, e := range elemSlice {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			log.Printf("malformed signature header: %q", signature)
			return false
		}
		elemMap[kv[0]] = kv[1]
	}

	signedPayload := fmt.Appendf(nil, "%s.%s", elemMap["t"], payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(signedPayload)
	if err != nil {
		log.Printf("error writing to hmac: %v", err)
		return false
	}
	expectedMAC := mac.Sum(nil)

	v1Header := []byte(elemMap["v1"])
	actualMAC := make([]byte, hex.DecodedLen(len(v1Header)))
	_, err = hex.Decode(actualMAC, v1Header)
	if err != nil {
		log.Printf("error decoding header: %v", err)
		return false
	}

	return hmac.Equal(expectedMAC, actualMAC)
}
