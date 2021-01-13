// Package util provides various useful functions for completing the
// "Whispering Gophers" code lab.
package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	rand2 "math/rand"
	"net"
	"time"
)

var seededRand *rand2.Rand = rand2.New(rand2.NewSource(time.Now().UnixNano()))

// Listen returns a Listener that listens on the first available port on the
// first available non-loopback IPv4 network interface.
func Listen() (net.Listener, error) {
	ip, err := ExternalIP()
	if err != nil {
		return nil, fmt.Errorf("could not find active non-loopback address: %v", err)
	}
	return net.Listen("tcp4", ip+":0")
}

func ExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// RandomID returns an 8 byte random string in hexadecimal.
func RandomID() string {
	b := make([]byte, 8)
	n, _ := rand.Read(b)
	return fmt.Sprintf("%x", b[:n])
}

func RandomString(length int, charset string) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
