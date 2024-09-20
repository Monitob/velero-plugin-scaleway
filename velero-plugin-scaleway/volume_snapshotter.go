/*
Copyright 2017, 2019 the Velero contributors.

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

package main

import (
	"context"

	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/sirupsen/logrus"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	regionKey    = "region"
	sbsCSIDriver = "sbs-default.csi.scaleway.com"
)

type VolumeSnapshotter struct {
	log logrus.FieldLogger
	scw *scw.Client
}

func newVolumeSnapshotter(logger logrus.FieldLogger) *VolumeSnapshotter {
	return &VolumeSnapshotter{log: logger}
}

func (s *VolumeSnapshotter) Init(config map[string]string) error {
	if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config, regionKey, credentialProfileKey, configPathKey); err != nil {
		return err
	}

	region := config[regionKey]
	configPath := config[configPathKey]
	profileName := config[credentialProfileKey]

	if region == "" {
		return errors.Errorf("missing %s in scw configuration", regionKey)
	}
	client, err := newClientBuilder(s.log).WithUserAgent(userAgentPrefix).WithEnvProfile().WithRegion(region).Build(configPath, profileName)
	if err != nil {
		return errors.WithStack(err)
	}

	s.scw = client
	return nil
}

func (s *VolumeSnapshotter) CreateVolumeFromSnapshot(snapshotID, volumeAZ string, iops uint32) (volumeID string, err error) {
	// describe the snapshot, so we can apply its tags to the volume
	blockAPI := block.NewAPI(s.scw)
	getSnapOutput, err := blockAPI.GetSnapshot(&block.GetSnapshotRequest{
		Zone:       scw.Zone(volumeAZ),
		SnapshotID: snapshotID,
	}, scw.WithContext(context.Background()))
	if err != nil {
		s.log.Infof("failed to describe snap shot: %v", err)

		return "", errors.WithStack(err)
	}

	if getSnapOutput != nil {
		return "", errors.Errorf("expected snapshot from GetSnapshot for %s", snapshotID)
	}

	// filter tags through getTagsForCluster() function in order to apply
	// proper ownership tags to restored volumes
	input := &block.CreateVolumeRequest{
		FromSnapshot: &block.CreateVolumeRequestFromSnapshot{
			SnapshotID: snapshotID,
		},
		Zone:     scw.Zone(volumeAZ),
		PerfIops: scw.Uint32Ptr(iops),
	}

	if len(getSnapOutput.Tags) > 0 {
		input.Tags = getTagsForCluster(getSnapOutput.Tags)
	}

	output, err := blockAPI.CreateVolume(input, scw.WithContext(context.Background()))
	if err != nil {
		return "", errors.WithStack(err)
	}

	return output.ID, nil
}

func (s *VolumeSnapshotter) GetVolumeInfo(volumeID, volumeAZ string) (string, *int64, error) {
	volumeInfo, err := s.describeVolume(volumeID)
	if err != nil {
		return "", nil, err
	}

	var (
		volumeType string
		iops64     int64
	)

	volumeType = volumeInfo.Type

	if volumeInfo.Specs != nil {
		specs := volumeInfo.Specs
		iops32 := specs.PerfIops
		iops64 = int64(*iops32)
	}

	return volumeType, &iops64, nil
}

func (s *VolumeSnapshotter) describeVolume(volumeID string) (block.Volume, error) {
	blockAPI := block.NewAPI(s.scw)

	input := &block.GetVolumeRequest{
		VolumeID: volumeID,
	}

	output, err := blockAPI.GetVolume(input, scw.WithContext(context.Background()))
	if err != nil {
		s.log.Infof("failed to describe snap shot: %v", err)

		return block.Volume{}, errors.WithStack(err)
	}

	if output == nil {
		return block.Volume{}, errors.Errorf("Expected one volume from DescribeVolumes for volume ID %v", volumeID)
	}

	return *output, nil
}

type tags []string

// method to get unique tags
func (t tags) unique() []string {
	uniqueTagsMap := make(map[string]bool)
	var uniqueTags []string

	for _, tag := range t {
		if !uniqueTagsMap[tag] {
			uniqueTagsMap[tag] = true
			uniqueTags = append(uniqueTags, tag)
		}
	}
	return uniqueTags
}

// Method to merge two []string slices and get unique tags
func (t tags) merge(other tags) []string {
	merged := append(t, other...)
	return tags(merged).unique()
}

// Method to convert a []string to the 'tags' type
func toTags(strSlice []string) tags {
	return tags(strSlice)
}

func (s *VolumeSnapshotter) CreateSnapshot(volumeID, snapshotName string, tags []string) (string, error) {
	// describe the volume, so we can copy its tags to the snapshot
	volumeInfo, err := s.describeVolume(volumeID)
	if err != nil {
		return "", err
	}

	blockAPI := block.NewAPI(s.scw)

	tagsFromVolume := toTags(volumeInfo.Tags)
	tagsMerged := tagsFromVolume.merge(tags)
	input := &block.CreateSnapshotRequest{
		VolumeID: volumeInfo.ID,
		Tags:     tagsMerged,
		Zone:     volumeInfo.Zone,
		Name:     fmt.Sprintf("vol-%s-snap-%s", volumeInfo.Name, snapshotName),
	}

	res, err := blockAPI.CreateSnapshot(input, scw.WithContext(context.Background()))
	if err != nil {
		return "", errors.WithStack(err)
	}

	return res.ID, nil
}

func getTagsForCluster(snapshotTags []string) []string {
	var result []string

	clusterName, haveSCWClusterNameEnvVar := os.LookupEnv("SCW_CLUSTER_NAME")

	if haveSCWClusterNameEnvVar {
		result = append(result, fmt.Sprintf("%s:%s", "kubernetes.io/cluster/"+clusterName, "owned"))
		result = append(result, fmt.Sprintf("%s:%s", "KubernetesCluster", clusterName))
	}

	for _, tag := range snapshotTags {
		if haveSCWClusterNameEnvVar && (strings.HasPrefix(tag, "kubernetes.io/cluster/") || tag == "KubernetesCluster") {
			// if the SCW_CLUSTER_NAME variable is found we want current cluster
			// to overwrite the old ownership on volumes
			continue
		}

		result = append(result, tag)
	}

	return result
}

func (s *VolumeSnapshotter) DeleteSnapshot(snapshotID string) error {
	blockAPI := block.NewAPI(s.scw)

	input := &block.DeleteSnapshotRequest{
		SnapshotID: snapshotID,
	}

	err := blockAPI.DeleteSnapshot(input, scw.WithContext(context.Background()))

	// if it's a NotFound error, we don't need to return an error
	// since the snapshot is not there.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if "InvalidSnapshot.NotFound" == apiErr.ErrorCode() {
			return nil
		}
	}

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

var ebsVolumeIDRegex = regexp.MustCompile("vol-.*")

func (s *VolumeSnapshotter) GetVolumeID(unstructuredPV runtime.Unstructured) (string, error) {
	pv := new(v1.PersistentVolume)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPV.UnstructuredContent(), pv); err != nil {
		return "", errors.WithStack(err)
	}
	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == sbsCSIDriver {
			return ebsVolumeIDRegex.FindString(pv.Spec.CSI.VolumeHandle), nil
		}
		s.log.Infof("Unable to handle CSI driver: %s", driver)
	}

	return "", nil
}

func (s *VolumeSnapshotter) SetVolumeID(unstructuredPV runtime.Unstructured, volumeID string) (runtime.Unstructured, error) {
	pv := new(v1.PersistentVolume)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPV.UnstructuredContent(), pv); err != nil {
		return nil, errors.WithStack(err)
	}
	if pv.Spec.CSI != nil {
		// PV is provisioned by CSI driver
		driver := pv.Spec.CSI.Driver
		if driver == sbsCSIDriver {
			pv.Spec.CSI.VolumeHandle = volumeID
		} else {
			return nil, fmt.Errorf("unable to handle CSI driver: %s", driver)
		}
	} else {
		return nil, errors.New("spec.csi not found")
	}

	res, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pv)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &unstructured.Unstructured{Object: res}, nil
}
