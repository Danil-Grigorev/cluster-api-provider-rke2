/*
Copyright 2023 SUSE.
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

// Package ignition aggregates all Ignition flavors into a single package to be consumed
// by the bootstrap provider by exposing an API similar to 'bootstrap/internal/ignition' package.
package ignition

import (
	"fmt"

	"github.com/rancher-sandbox/cluster-api-provider-rke2/bootstrap/internal/cloudinit"
	"github.com/rancher-sandbox/cluster-api-provider-rke2/bootstrap/internal/ignition/butane"

	ignitionTypes "github.com/coreos/ignition/v2/config/v3_3/types"
)

const (
	airGappedControlPlaneCommand = "INSTALL_RKE2_ARTIFACT_PATH=/opt/rke2-artifacts sh /opt/install.sh"
	controlPlaneCommand          = "curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=%[1]s sh -s - server"
	airGappedWorkerCommand       = "INSTALL_RKE2_ARTIFACT_PATH=/opt/rke2-artifacts INSTALL_RKE2_TYPE=\"agent\" sh /opt/install.sh"
	workerCommand                = "curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=%[1]s INSTALL_RKE2_TYPE=\"agent\" sh -s -"
)

var (
	serverSystemdServices = []string{
		"systemctl enable rke2-server.service",
		"systemctl start rke2-server.service",
	}

	workerSystemdServices = []string{
		"systemctl enable rke2-agent.service",
		"systemctl start rke2-agent.service",
	}
)

// JoinWorkerInput defines the context to generate a node user data.
type JoinWorkerInput struct {
	*cloudinit.BaseUserData

	Ignition ignitionTypes.Config
}

// ControlPlaneInput defines the context to generate a controlplane instance user data.
type ControlPlaneInput struct {
	*cloudinit.ControlPlaneInput

	Ignition ignitionTypes.Config
}

// NewJoinWorker returns Ignition configuration for new worker node joining the cluster.
func NewJoinWorker(input *JoinWorkerInput) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("input can't be nil")
	}

	if input.BaseUserData == nil {
		return nil, fmt.Errorf("base userdata can't be nil")
	}

	deployRKE2Command, err := getWorkerRKE2Commands(input.BaseUserData)
	if err != nil {
		return nil, fmt.Errorf("failed to get rke2 command: %w", err)
	}

	input.DeployRKE2Commands = deployRKE2Command
	input.WriteFiles = append(input.WriteFiles, input.ConfigFile)

	return butane.Render(input.BaseUserData, input.Ignition)
}

// NewJoinControlPlane returns Ignition configuration for new controlplane node joining the cluster.
func NewJoinControlPlane(input *ControlPlaneInput) ([]byte, error) {
	processedInput, err := controlPlaneConfigInput(input)
	if err != nil {
		return nil, fmt.Errorf("failed to process controlplane input: %w", err)
	}

	return butane.Render(&processedInput.BaseUserData, input.Ignition)
}

// NewInitControlPlane returns Ignition configuration for bootstrapping new cluster.
func NewInitControlPlane(input *ControlPlaneInput) ([]byte, error) {
	processedInput, err := controlPlaneConfigInput(input)
	if err != nil {
		return nil, fmt.Errorf("failed to process controlplane input: %w", err)
	}

	return butane.Render(&processedInput.BaseUserData, input.Ignition)
}

func controlPlaneConfigInput(input *ControlPlaneInput) (*ControlPlaneInput, error) {
	if input == nil {
		return nil, fmt.Errorf("input can't be nil")
	}

	if input.ControlPlaneInput == nil {
		return nil, fmt.Errorf("controlplane input can't be nil")
	}

	deployRKE2Command, err := getControlPlaneRKE2Commands(&input.BaseUserData)
	if err != nil {
		return nil, fmt.Errorf("failed to get rke2 command: %w", err)
	}

	input.DeployRKE2Commands = deployRKE2Command
	input.WriteFiles = append(input.WriteFiles, input.Certificates.AsFiles()...)
	input.WriteFiles = append(input.WriteFiles, input.ConfigFile)

	return input, nil
}

func getControlPlaneRKE2Commands(baseUserData *cloudinit.BaseUserData) ([]string, error) {
	return getRKE2Commands(baseUserData, controlPlaneCommand, airGappedControlPlaneCommand, serverSystemdServices)
}

func getWorkerRKE2Commands(baseUserData *cloudinit.BaseUserData) ([]string, error) {
	return getRKE2Commands(baseUserData, workerCommand, airGappedWorkerCommand, workerSystemdServices)
}

func getRKE2Commands(baseUserData *cloudinit.BaseUserData, command, airgappedCommand string, systemdServices []string) ([]string, error) {
	if baseUserData == nil {
		return nil, fmt.Errorf("base user data can't be nil")
	}

	if baseUserData.RKE2Version == "" {
		return nil, fmt.Errorf("rke2 version can't be empty")
	}

	rke2Commands := []string{}

	if baseUserData.AirGapped {
		rke2Commands = append(rke2Commands, airgappedCommand)
	} else {
		rke2Commands = append(rke2Commands, fmt.Sprintf(command, baseUserData.RKE2Version))
	}

	rke2Commands = append(rke2Commands, systemdServices...)

	return rke2Commands, nil
}
