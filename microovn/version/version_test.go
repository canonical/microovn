package version

import "testing"

func TestMajorVersionNoSeparator(t *testing.T) {
	inputVersion := "versionWithNoSeparator"
	result := MajorVersion(inputVersion)
	if result != inputVersion {
		t.Fatalf("MajorVersion(%s) = %s, expected %s",
			inputVersion, result, inputVersion)
	}
}

func TestMajorVersionShort(t *testing.T) {
	inputVersion := "24.03"
	expect := "24.03"
	result := MajorVersion(inputVersion)
	if result != expect {
		t.Fatalf("Majorversion(%s) = %s, expected %s",
			inputVersion, result, expect)
	}
}

func TestMajorVersionSemVer(t *testing.T) {
	inputVersion := "24.03.0"
	expect := "24.03"
	result := MajorVersion(inputVersion)
	if result != expect {
		t.Fatalf("Majorversion(%s) = %s, expected %s",
			inputVersion, result, expect)
	}
}
