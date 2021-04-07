package virtualbox

import (
	"crypto/rand"

	"github.com/pkg/errors"

	"fmt"
)

// generateLocalUnicastMac does generate local unicast mac address
// https://stackoverflow.com/questions/21018729/generate-mac-address-in-go
func generateLocalUnicastMac() (string, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		fmt.Println("error:", err)
		return "", errors.Wrap(err, "could not read random")
	}
	// Ensure local mac address bit#1 set to 1 (|2)
	// Ensure unicat address --> but#0 set to 0 (&buf[0] = (buf[0] | 2) & 0xfe)
	buf[0] = (buf[0] | 2) & 0xfe
	addr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x\n", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	return addr, nil
}
