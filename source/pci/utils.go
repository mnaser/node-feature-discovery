/*
Copyright 2020-2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pci

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/node-feature-discovery/pkg/api/feature"
	"sigs.k8s.io/node-feature-discovery/source"
)

var mandatoryDevAttrs = []string{"class", "vendor", "device", "subsystem_vendor", "subsystem_device"}
var optionalDevAttrs = []string{"sriov_totalvfs", "iommu_group/type"}

// Read a single PCI device attribute
// A PCI attribute in this context, maps to the corresponding sysfs file
func readSinglePciAttribute(devPath string, attrName string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(devPath, attrName))
	if err != nil {
		return "", fmt.Errorf("failed to read device attribute %s: %v", attrName, err)
	}
	// Strip whitespace and '0x' prefix
	attrVal := strings.TrimSpace(strings.TrimPrefix(string(data), "0x"))

	if attrName == "class" && len(attrVal) > 4 {
		// Take four first characters, so that the programming
		// interface identifier gets stripped from the raw class code
		attrVal = attrVal[0:4]
	}
	return attrVal, nil
}

// Read information of one PCI device
func readPciDevInfo(devPath string) (*feature.InstanceFeature, error) {
	attrs := make(map[string]string)
	for _, attr := range mandatoryDevAttrs {
		attrVal, err := readSinglePciAttribute(devPath, attr)
		if err != nil {
			return nil, fmt.Errorf("failed to read device %s: %s", attr, err)
		}
		attrs[attr] = attrVal
	}
	for _, attr := range optionalDevAttrs {
		attrVal, err := readSinglePciAttribute(devPath, attr)
		if err == nil {
			attrs[attr] = attrVal
		}
	}
	return feature.NewInstanceFeature(attrs), nil
}

// detectPci detects available PCI devices and retrieves their device attributes.
// An error is returned if reading any of the mandatory attributes fails.
func detectPci() ([]feature.InstanceFeature, error) {
	sysfsBasePath := source.SysfsDir.Path("bus/pci/devices")

	devices, err := ioutil.ReadDir(sysfsBasePath)
	if err != nil {
		return nil, err
	}

	// Iterate over devices
	devInfo := make([]feature.InstanceFeature, 0, len(devices))
	for _, device := range devices {
		info, err := readPciDevInfo(filepath.Join(sysfsBasePath, device.Name()))
		if err != nil {
			klog.Error(err)
			continue
		}
		devInfo = append(devInfo, *info)
	}

	return devInfo, nil
}
