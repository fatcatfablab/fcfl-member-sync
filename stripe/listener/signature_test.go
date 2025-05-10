package listener

import (
	"testing"
)

func TestVerifySignature(t *testing.T) {
	for _, tt := range []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		err       bool
	}{
		{
			name:      "Malformed signature 0",
			payload:   []byte{},
			signature: "malformed signature",
			secret:    "",
			err:       true,
		},
		{
			name:      "Malformed signature 1",
			payload:   []byte{},
			signature: "v1=xxx",
			secret:    "",
			err:       true,
		},
		{
			name:      "Valid signature 0",
			payload:   []byte{},
			signature: "t=1746842775,v1=9aca2217698466654f05c35cd53c24e7ec0951a76b3c5f878f8d7bf3467bce69",
			secret:    "",
			err:       false,
		},
		{
			name:      "Valid signature 1",
			payload:   []byte{},
			signature: "t=1746842775,v1=355dce2831e39aea16cd3cd7d37e47f86711f2799bd0949e61c514420d97947d",
			secret:    "secret",
			err:       false,
		},
		{
			name:      "Valid signature 2",
			payload:   []byte{255, 255},
			signature: "t=1746842775,v1=c713c607a21429531202ac03ff5de580ed0987b15e3db94b0edbec6cd1212dfe",
			secret:    "secret",
			err:       false,
		},
		{
			name:      "Invalid signature 0",
			payload:   []byte{255, 255},
			signature: "t=1746842775,v1=deadbeefffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			secret:    "secret",
			err:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifySignature(tt.payload, tt.signature, tt.secret); (err != nil) != tt.err {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
