package pandoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMetadata_RelativePathRepro(t *testing.T) {
	// 1. Setup a directory structure
	// tmp/
	//   project/
	//     input.md
	//     cover.jpg
	tmpDir, err := os.MkdirTemp("", "repro_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatal(err)
	}

	coverFile := filepath.Join(projectDir, "cover.jpg")
	if err := os.WriteFile(coverFile, []byte("fake image"), 0600); err != nil {
		t.Fatal(err)
	}

	// 2. We are currently running in the package directory or some other dir.
	// We want to simulate processing input.md which refers to "cover.jpg"
	meta := map[string]interface{}{
		"epub-cover-image": "cover.jpg",
	}

	// 3. New ValidateMetadata (with baseDir) checks relative to baseDir.
	// We pass projectDir as the baseDir since input.md would be there.

	err = ValidateMetadata(meta, projectDir)

	if err != nil {
		t.Errorf("Expected nil error when checking relative path with baseDir, but got: %v", err)
	} else {
		t.Log("Successfully validated relative path with baseDir")
	}
}
