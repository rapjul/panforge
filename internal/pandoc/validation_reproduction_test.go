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

	// 3. Current ValidateMetadata (without baseDir) checks relative to CWD.
	// Since we are NOT in projectDir, this should fail if the bug exists.
	// Note: If we were lucky and CWD happened to have cover.jpg, it would pass, but highly unlikely in test env.

	err = ValidateMetadata(meta)

	// The user says "command says the cover image is not found", so we EXPECT an error here currently.
	// If the code was correct (resolving relative to input file), we'd need to pass the baseDir.
	// But since the current code DOES NOT take baseDir, it looks for "cover.jpg" in CWD.
	// So we expect this to fail.

	if err == nil {
		t.Error("Expected error because cover.jpg is not in CWD, but got nil")
	} else {
		t.Logf("Got expected error (reproduction success): %v", err)
	}
}
