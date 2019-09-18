package main

import (
	"fmt"
	"net"
	"strings"
)

func Ips() (map[string][]string, error) {

	ips :=  make(map[string][]string)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		byName, err := net.InterfaceByName(i.Name)
		if err != nil {
			return nil, err
		}
		addresses, err := byName.Addrs()
		for _, v := range addresses {
			ips[byName.Name] = append(ips[byName.Name],v.String())
		}
	}
	return ips, nil
}

func FindInterface(ip string) (string,error) {
	ips,err := Ips()
	if err != nil {
		return "",err
	}
	// if no ip given, return the first device with internal ip
	if ip == "" {
		for inter,addrs := range ips {
			for _,addr := range addrs {
				if strings.HasPrefix(addr,"10.") {
					return inter,nil
				}
			}
		}
	}
	for inter,addrs := range ips {
		for _,addr := range addrs {
			if strings.HasPrefix(addr,ip) {
				return inter,nil
			}
		}
	}
	return "",fmt.Errorf("interface not found")
}