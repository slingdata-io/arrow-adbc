// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package drivermgr

import (
	"os"
	"strings"
	"testing"
)

// TestLoadDriverManagerLibrary guards against regressions in registerFunctions,
// which binds every ADBC symbol in one shot. A single binding purego rejects on
// a given platform panics the whole process before any driver is touched.
//
// This is the exact failure behind the Windows report: AdbcErrorGetDetail
// returns a struct by value, and purego only supports struct returns on
// darwin/linux, so registration panicked with
// "purego: struct arguments are only supported on darwin and linux".
//
// Reported downstream as slingdata-io/sling-cli#783.
//
// Requires the driver manager library to be present; set ADBC_DRIVER_MANAGER_LIB
// to point at it explicitly.
func TestLoadDriverManagerLibrary(t *testing.T) {
	// A panic here fails the test rather than taking down the run, so the
	// message lands in the test output where it can be read.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registering ADBC functions panicked: %v", r)
		}
	}()

	lib, err := loadDriverManagerLibrary()
	if err != nil {
		if isLibraryMissing(err) {
			t.Skipf("ADBC driver manager library not installed: %v", err)
		}
		t.Fatalf("failed to load ADBC driver manager library: %v", err)
	}

	if lib == 0 {
		t.Fatal("loaded ADBC driver manager library, got a null handle")
	}

	// Registration populates these; a nil binding means a symbol silently
	// failed to resolve.
	if adbcDatabaseNew == nil {
		t.Error("AdbcDatabaseNew was not bound")
	}
	if adbcConnectionNew == nil {
		t.Error("AdbcConnectionNew was not bound")
	}
	if adbcStatementNew == nil {
		t.Error("AdbcStatementNew was not bound")
	}
}

// isLibraryMissing reports whether err is the library being absent, as opposed
// to it being present but unusable. Only the former is skippable.
func isLibraryMissing(err error) bool {
	if os.Getenv("ADBC_DRIVER_MANAGER_LIB") != "" {
		// Explicitly pointed at a library, so any failure is a real one.
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"no such file",
		"image not found",
		"cannot open shared object file",
		"not found",
		"could not be found",
		"specified module",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
