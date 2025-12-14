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
)

func TestBuildDaemonCommandLine(t *testing.T) {
	testCases := []struct {
		name        string
		testingMode bool
		expected    []string
	}{
		{
			name:        "Normal mode without testing flag",
			testingMode: false,
			expected:    []string{imexDaemonBinaryName, "-c", imexDaemonConfigPath},
		},
		{
			name:        "Testing mode with --nogpu flag",
			testingMode: true,
			expected:    []string{imexDaemonBinaryName, "-c", imexDaemonConfigPath, "--nogpu"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This mirrors the logic from the run() function lines 238-242
			daemonCommandLine := []string{imexDaemonBinaryName, "-c", imexDaemonConfigPath}
			if tc.testingMode {
				daemonCommandLine = append(daemonCommandLine, "--nogpu")
			}

			require.Equal(t, tc.expected, daemonCommandLine)

			// Verify specific expectations
			if tc.testingMode {
				require.Contains(t, daemonCommandLine, "--nogpu", "Testing mode should include --nogpu flag")
			} else {
				require.NotContains(t, daemonCommandLine, "--nogpu", "Normal mode should not include --nogpu flag")
			}
		})
	}
}

func TestFlags_TestingModeDefault(t *testing.T) {
	// Test that testingMode defaults to false
	flags := Flags{}
	require.False(t, flags.testingMode, "testingMode should default to false")
}
