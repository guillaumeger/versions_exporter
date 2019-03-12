package main

import (
	"os"
	"testing"
)

func TestGetRefreshIntervalEnv(t *testing.T) {
	os.Setenv("VERSIONS_EXPORTER_REFRESH_INTERVAL", "2h")
	v := getRefreshInterval()
	if v != "2h" {
		t.Errorf("Expected: 2h, got: %v\n", v)
	}
}

func TestGetRefreshIntervalDefault(t *testing.T) {
	os.Unsetenv("VERSIONS_EXPORTER_REFRESH_INTERVAL")
	v := getRefreshInterval()
	if v != "1h" {
		t.Errorf("Expected: 2h, got: %v\n", v)
	}
}

func TestTestGetAnnotationNameEnv(t *testing.T) {
	os.Setenv("VERSIONS_EXPORTER_ANNOTATION_NAME", "patate/poil")
	v := getAnnotationName()
	if v != "patate/poil" {
		t.Errorf("Expected: patate/poil, got: %v\n", v)
	}
}

func TestTestGetAnnotationNameDefault(t *testing.T) {
	os.Unsetenv("VERSIONS_EXPORTER_ANNOTATION_NAME")
	v := getAnnotationName()
	if v != "versions_exporter/githubRepo" {
		t.Errorf("Expected: versions_exporter/githubRepo, got: %v\n", v)
	}
}

func TestTestGetPortEnv(t *testing.T) {
	os.Setenv("VERSIONS_EXPORTER_PORT", "8080")
	v := getPort()
	if v != "8080" {
		t.Errorf("Expected: 8080, got: %v\n", v)
	}
}

func TestTestGetPortDefault(t *testing.T) {
	os.Unsetenv("VERSIONS_EXPORTER_PORT")
	v := getPort()
	if v != "8083" {
		t.Errorf("Expected: 8083, got: %v\n", v)
	}
}
