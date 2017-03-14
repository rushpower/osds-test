// Copyright (c) 2016 Huawei Technologies Co., Ltd. All Rights Reserved.
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package utils

import (
	"errors"
	"log"
	"net"
	"strings"
)

// getLocalIP returns the non loopback local IP of the host
func getLocalIP() ([]string, error) {
	var ipList []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ipList, err
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipList = append(ipList, ipnet.IP.String())
			}
		}
	}

	if len(ipList) == 0 {
		err = errors.New("Can't find host ip!")
		return ipList, err
	}

	return ipList, nil

}

// Get preferred outbound ip of this machine
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")

	return localAddr[0:idx], nil
}

func GetHostIP() (string, error) {
	hostIp, err := getOutboundIP()
	if err != nil {
		log.Println("Can't get host ip:", err)
		return "", err
	}

	return hostIp, nil
}
