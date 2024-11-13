package core

import (
	"fmt"
	"net/netip"

	"github.com/3th1nk/cidr"
)

func convertIP(ip string) (res []byte, err error) {
	pa, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	res = pa.AsSlice()
	return res, err
}

func convertCIDR(iprange string, ipv4MaxLimit int, ipv6MaxLimit int) (upperres []byte, lowerres []byte, err error) {
	cp, err := cidr.Parse(iprange)
	if err != nil {
		return nil, nil, err
	}
	if cp.IsIPv4() {
		ones, _ := cp.MaskSize()
		if ones < ipv4MaxLimit {
			return nil, nil, fmt.Errorf("IPv4 mask limit reach for range %s (max required %d), ignoring", iprange, ipv4MaxLimit)
		}
	}
	if cp.IsIPv6() {
		ones, _ := cp.MaskSize()
		if ones < ipv6MaxLimit {
			return nil, nil, fmt.Errorf("IPv6 mask limit reach for range %s (max required %d), ignoring", iprange, ipv6MaxLimit)
		}
	}
	upper, _ := netip.AddrFromSlice(cp.Broadcast())
	lower, _ := netip.AddrFromSlice(cp.Network())
	return upper.AsSlice(), lower.AsSlice(), nil
}
