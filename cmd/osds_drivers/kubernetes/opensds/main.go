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

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"
)

const (
	URL_PREFIX string = "http://10.2.1.233:8080"
	// LINK_PREFIX   string = "/dev/cinder-volumes/volume-"
	CINDER_LINK_PREFIX string = "/dev/mapper/cinder--volumes-volume--"
	MANILA_LINK_PREFIX string = "/var/lib/manila/mnt/share-"
	DEVICE_PREFIX      string = "/dev/"
)

type OpenSDSOptions struct {
	DefaultOptions
	VolumeId     string `json:"volumeId"`
	ResourceType string `json:"resourceType"`
}

type OpenSDSPlugin struct{}

func (OpenSDSPlugin) Init() Result {
	return Succeed()
}

func (OpenSDSPlugin) NewOptions() interface{} {
	var option = &OpenSDSOptions{}
	return option
}

func (OpenSDSPlugin) Attach(opts interface{}) Result {
	opt := opts.(*OpenSDSOptions)

	switch opt.ResourceType {
	case "cinder":
		return cinderAttach(opt)
	case "manila":
		return manilaAttach(opt)
	default:
		err := errors.New("Backend resource not supported!")
		return Fail(err.Error())
	}
}

func cinderAttach(opt *OpenSDSOptions) Result {
	volId := opt.VolumeId
	url := URL_PREFIX + "/api/v1/volumes/action/cinder/" + volId

	linkPath := CINDER_LINK_PREFIX + strings.Replace(volId, "-", "--", 4)
	path, err := generateDevicePath(linkPath)
	if err != nil {
		return Fail(err.Error())
	}

	// fmt.Println("Start POST request to attach volume, url =", url)

	req := httplib.Post(url).SetTimeout(100*time.Second, 50*time.Second)

	var volumeRequest VolumeRequest
	volumeRequest.ActionType = "attach"
	volumeRequest.Device = path

	req.JSONBody(volumeRequest)
	resp, err := req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	err = CheckHTTPResponseStatusCode(resp)
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	if strings.Contains(string(rbody), "Success") {
		return Result{
			Status: "Success",
			Device: path,
		}
	} else {
		err = errors.New("Attach volume failed!")
		return Fail(err.Error())
	}
}

func manilaAttach(opt *OpenSDSOptions) Result {
	shrId := opt.VolumeId
	url := URL_PREFIX + "/api/v1/shares/action/manila/" + shrId

	req := httplib.Get(url).SetTimeout(100*time.Second, 50*time.Second)

	resp, err := req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	sdr := new(ShareDetailResponse)
	err = json.Unmarshal(rbody, sdr)
	if err != nil {
		return Fail(err.Error())
	}

	if sdr.ExportLocation == "" {
		err = errors.New("Share not exported!")
		return Fail(err.Error())
	} else {
		return Result{
			Status: "Success",
			Device: sdr.ExportLocation,
		}
	}
}

func (OpenSDSPlugin) Detach(device string) Result {
	if !strings.HasPrefix(device, CINDER_LINK_PREFIX) {
		if strings.HasPrefix(device, MANILA_LINK_PREFIX) {
			return Succeed()
		} else {
			err := errors.New("Expect device prefix: " + CINDER_LINK_PREFIX)
			return Fail(err.Error())
		}
	}

	volumeId := device[len(CINDER_LINK_PREFIX):len(device)]
	volId := strings.Replace(volumeId, "--", "-", 4)

	url := URL_PREFIX + "/api/v1/volumes/cinder/" + volId

	// fmt.Println("Start GET request to get volume, url =", url)

	req := httplib.Get(url).SetTimeout(100*time.Second, 50*time.Second)

	resp, err := req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	volumeResponse := new(VolumeResponse)
	err = json.Unmarshal(rbody, volumeResponse)
	if err != nil {
		return Fail(err.Error())
	}

	if volumeResponse.Status != "in-use" {
		err = errors.New("The status of volume is not in-use!")
		return Fail(err.Error())
	}

	url = URL_PREFIX + "/api/v1/volumes/action/cinder/" + volId

	// fmt.Println("Start POST request to detach volume, url =", url)

	req = httplib.Post(url).SetTimeout(10*time.Second, 5*time.Second)

	var volumeRequest VolumeRequest
	volumeRequest.ActionType = "detach"
	volumeRequest.Attachment = volumeResponse.Attachments[0]["attachment_id"]

	req.JSONBody(volumeRequest)
	resp, err = req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	err = CheckHTTPResponseStatusCode(resp)
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	if strings.Contains(string(rbody), "Success") {
		return Succeed()
	} else {
		err = errors.New("Detach volume failed!")
		return Fail(err.Error())
	}
}

func (OpenSDSPlugin) Mount(mountDir string, device string, opts interface{}) Result {
	opt := opts.(*OpenSDSOptions)

	volId := opt.VolumeId
	url := URL_PREFIX + "/api/v1/volumes/action/cinder/" + volId

	// fmt.Println("Start POST request to mount volume, url =", url)

	req := httplib.Post(url).SetTimeout(100*time.Second, 50*time.Second)

	var volumeRequest VolumeRequest
	volumeRequest.ActionType = "mount"
	volumeRequest.MountDir = mountDir
	volumeRequest.Device = device
	volumeRequest.FsType = opt.FsType

	req.JSONBody(volumeRequest)
	resp, err := req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	err = CheckHTTPResponseStatusCode(resp)
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	if strings.Contains(string(rbody), "Success") {
		return Succeed()
	} else {
		err = errors.New("Mount volume failed!")
		return Fail(err.Error())
	}
}

func (OpenSDSPlugin) Unmount(mountDir string) Result {
	volId := "null"
	url := URL_PREFIX + "/api/v1/volumes/action/cinder/" + volId

	// fmt.Println("Start POST request to unmount volume, url =", url)

	req := httplib.Post(url).SetTimeout(100*time.Second, 50*time.Second)

	var volumeRequest VolumeRequest
	volumeRequest.ActionType = "unmount"
	volumeRequest.MountDir = mountDir

	req.JSONBody(volumeRequest)
	resp, err := req.Response()
	if err != nil {
		return Fail(err.Error())
	}

	err = CheckHTTPResponseStatusCode(resp)
	if err != nil {
		return Fail(err.Error())
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fail(err.Error())
	}
	if strings.Contains(string(rbody), "Success") {
		return Succeed()
	} else {
		err = errors.New("Unmount volume failed!")
		return Fail(err.Error())
	}
}

func main() {
	RunPlugin(&OpenSDSPlugin{})
}

func generateDevicePath(link string) (string, error) {
	path, err := os.Readlink(link)
	if err != nil {
		err = errors.New("Can't find device path!")
		return "", err
	}

	slice := strings.Split(path, "/")
	device := "/dev/" + slice[1]
	return device, nil
}
