/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package discover

import (
	"fmt"
	"path/filepath"

	"tags.cncf.io/container-device-interface/pkg/cdi"
)

var _ Discover = (*Hook)(nil)

// Devices returns an empty list of devices for a Hook discoverer.
func (h Hook) Devices() ([]Device, error) {
	return nil, nil
}

// Mounts returns an empty list of mounts for a Hook discoverer.
func (h Hook) Mounts() ([]Mount, error) {
	return nil, nil
}

// Hooks allows the Hook type to also implement the Discoverer interface.
// It returns a single hook
func (h Hook) Hooks() ([]Hook, error) {
	return []Hook{h}, nil
}

// CreateCreateSymlinkHook creates a hook which creates a symlink from link -> target.
func CreateCreateSymlinkHook(nvidiaCDIHookPath string, links []string) Discover {
	if len(links) == 0 {
		return None{}
	}

	var args []string
	for _, link := range links {
		args = append(args, "--link", link)
	}
	return CreateNvidiaCDIHook(
		nvidiaCDIHookPath,
		"create-symlinks",
		args...,
	)
}

// CreateNvidiaCDIHook creates a hook which invokes the NVIDIA Container CLI hook subcommand.
func CreateNvidiaCDIHook(nvidiaCDIHookPath string, hookName string, additionalArgs ...string) Hook {
	return cdiHook(nvidiaCDIHookPath).Create(hookName, additionalArgs...)
}

type cdiHook string

func (c cdiHook) Create(name string, args ...string) Hook {
	return Hook{
		Lifecycle: cdi.CreateContainerHook,
		Path:      string(c),
		Args:      append(c.requiredArgs(name), args...),
		Env:       []string{fmt.Sprintf("NVIDIA_CTK_DEBUG=%v", false)},
	}
}
func (c cdiHook) requiredArgs(name string) []string {
	base := filepath.Base(string(c))
	if base == "nvidia-ctk" {
		return []string{base, "hook", name}
	}
	return []string{base, name}
}
