package jsonDNS

import (
	"fmt"
	"net"
	"testing"
)

func TestFindIp(t *testing.T) {

	fmt.Println(IsGlobalIP(net.IP{127, 0, 0, 1}))
	fmt.Println(IsGlobalIP(net.IP{192, 168, 0, 0}))
	fmt.Println(IsGlobalIP(net.IP{110, 100, 100, 100}))

}
