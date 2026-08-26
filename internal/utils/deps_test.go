package utils_test

import (
	"testing"

	"github.com/rapjul/panforge/internal/utils"
)

func TestCheckTool(t *testing.T) {
	// 1. Test existing tool (go should be present in test env)
	res := utils.CheckTool("go", "--version")
	if !res.Found {
		t.Error("CheckTool('go') returned found=false, expected true")
	}

	// 2. Test non-existing tool
	res = utils.CheckTool("non_existent_tool_xyz_123", "")
	if res.Found {
		t.Error("CheckTool('non_existent_tool...') returned found=true, expected false")
	}
}

func TestCheckPandoc(t *testing.T) {
	// Should return boolean without panic
	// Value depends on env, but we just want coverage of the function call
	_ = utils.CheckPandoc()
}

func TestCheckPDFEngine(t *testing.T) {
	// Try checking a known pdf engine if available, or just verify logic
	// We'll trust CheckTool logic coverage mostly, but exercise the wrapper
	_ = utils.CheckPDFEngine("pdflatex")
}

// TestCheckOptionalTool tests the warning check for optional tools.
func TestCheckOptionalTool(t *testing.T) {
	// Should log warning if missing but not fail/panic
	utils.CheckOptionalTool("non_existent_tool_abc")
}

// TestCheckTypst verifies CheckTypst function execution.
func TestCheckTypst(t *testing.T) {
	// Should return boolean without panic
	_ = utils.CheckTypst()
}
