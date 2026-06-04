package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadContainerImageFile(t *testing.T) {
	content := "# core services\ndocker.io/library/python:latest\n\n  docker.io/library/nginx:latest  \n\t# trailing comment\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "images.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readContainerImageFile(path)
	if err != nil {
		t.Fatalf("readContainerImageFile failed: %v", err)
	}

	want := []string{
		"docker.io/library/python:latest",
		"docker.io/library/nginx:latest",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readContainerImageFile() = %v, want %v", got, want)
	}
}

func TestReadContainerImageFile_Missing(t *testing.T) {
	if _, err := readContainerImageFile(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
