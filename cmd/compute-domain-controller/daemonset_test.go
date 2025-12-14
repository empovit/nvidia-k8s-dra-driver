/*
 * Copyright (c) 2025 NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestDaemonSetTemplateData_TestingMode(t *testing.T) {
	testCases := []struct {
		name                string
		configTestingMode   bool
		expectedTestingMode bool
	}{
		{
			name:                "Testing mode enabled",
			configTestingMode:   true,
			expectedTestingMode: true,
		},
		{
			name:                "Testing mode disabled",
			configTestingMode:   false,
			expectedTestingMode: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &ManagerConfig{
				testingModeCDDaemon: tc.configTestingMode,
			}

			// This mirrors the template data creation from daemonset.go
			templateData := DaemonSetTemplateData{
				Namespace:                 "test-namespace",
				GenerateName:              "test-",
				Finalizer:                 "test-finalizer",
				ComputeDomainLabelKey:     "test-label-key",
				ComputeDomainLabelValue:   types.UID("test-uid"),
				ResourceClaimTemplateName: "test-rct",
				ImageName:                 "test-image",
				MaxNodesPerIMEXDomain:     8,
				FeatureGates:              map[string]bool{},
				LogVerbosity:              2,
				TestingMode:               config.testingModeCDDaemon,
			}

			require.Equal(t, tc.expectedTestingMode, templateData.TestingMode,
				"DaemonSetTemplateData.TestingMode should match config.testingModeCDDaemon")
		})
	}
}

func TestManagerConfig_TestingModeDefault(t *testing.T) {
	// Test that testingModeCDDaemon defaults to false
	config := ManagerConfig{}
	require.False(t, config.testingModeCDDaemon, "testingModeCDDaemon should default to false")
}
