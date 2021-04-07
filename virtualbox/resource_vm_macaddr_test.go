package virtualbox

import (
	"bytes"
	"crypto/rand"
	"fmt"
)

func Example_generateLocalUnicastMac() {
	originalReader := rand.Reader
	defer func() { rand.Reader = originalReader }()

	rand.Reader = bytes.NewBuffer([]byte{0x16, 0x2e, 0x99, 0x92, 0x26, 0x13})
	got, _ := generateLocalUnicastMac()
	fmt.Printf("%s", got)
	// Output: 16:2e:99:92:26:13
}
