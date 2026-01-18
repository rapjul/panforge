package pandoc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"reflect"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/options"
	"github.com/rapjul/panforge/internal/utils"
)

var internalFlags map[string]bool

func init() {
	internalFlags = make(map[string]bool)
	val := options.Options{}
	t := reflect.TypeOf(val)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		flagName := field.Tag.Get("flag")
		if flagName != "" {
			internalFlags["--"+flagName] = true
		}
		shorthand := field.Tag.Get("shorthand")
		if shorthand != "" {
			internalFlags["-"+shorthand] = true
		}
	}
	// Explicitly ignore help flags as they are handled by Cobra
	internalFlags["--help"] = true
	internalFlags["-h"] = true
}

// NormalizeFormat strips extensions like +extension or -extension from the format string.
//
// Parameters:
//   - `spec`: the format string (e.g., "markdown+yaml_metadata_block")
//
// Returns:
//   - string: the base format (e.g., "markdown")
func NormalizeFormat(spec string) string {
	parts := strings.FieldsFunc(spec, func(r rune) bool {
		return r == '+' || r == '-'
	})
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// ExtForFormat returns the file extension for a given pandoc format.
//
// Parameters:
//   - `fmtStr`: the pandoc format string (e.g. "latex")
//
// Returns:
//   - string: the corresponding file extension (e.g. "tex")
func ExtForFormat(fmtStr string) string {
	fmtStr = strings.ToLower(fmtStr)
	switch fmtStr {
	case "html", "html5":
		return "html"
	case "epub", "epub3":
		return "epub"
	case "docx":
		return "docx"
	case "markdown", "md":
		return "md"
	case "latex", "tex":
		return "tex"
	case "pdf":
		return "pdf"
	case "beamer":
		return "pdf" // Simplified assumption, often pdf output
	default:
		return fmtStr
	}
}

// GetSupportedFormats queries pandoc for supported formats.
//
// Returns:
//   - []string: a slice of supported format names
//   - error: any error encountered (e.g. pandoc not found)
func GetSupportedFormats() ([]string, error) {
	cmd := exec.Command("pandoc", "--list-output-formats")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		lines := strings.Split(string(out), "\n")
		var formats []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				formats = append(formats, strings.TrimSpace(line))
			}
		}
		return formats, nil
	}
	return []string{}, nil // Fallback or empty if not found
}

// GenerateOutputFilename logic determines the output filename based on configuration.
//
// Parameters:
//   - `inputFile`: path to the input file
//   - `cfg`: global configuration
//   - `metaOut`: format-specific configuration from YAML
//   - `pandocFmt`: target pandoc format
//
// Returns:
//   - string: the generated filename
func GenerateOutputFilename(inputFile string, cfg *config.Config, metaOut map[string]interface{}, pandocFmt string) string {
	if val, ok := metaOut["output"]; ok {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}

	title := cfg.Title
	if title == "" {
		// try to read title from first heading of input file
		content, _ := os.ReadFile(inputFile) // ignore error
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				break
			}
		}
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	}

	// Template
	tmpl := cfg.FilenameTemplate
	if tmpl == "" {
		// Default
		tmpl = "{title}_{date}.{ext}"
	}

	// Variables
	dateStr := utils.FormatDate()
	timeStr := time.Now().Format("15-04-05")
	ext := ExtForFormat(pandocFmt)
	author := cfg.Author

	// Substitution
	result := strings.ReplaceAll(tmpl, "{date}", dateStr)
	result = strings.ReplaceAll(result, "{time}", timeStr)
	result = strings.ReplaceAll(result, "{title}", title)
	result = strings.ReplaceAll(result, "{author}", author)
	result = strings.ReplaceAll(result, "{title-slug}", utils.Slugify(title))
	result = strings.ReplaceAll(result, "{author-slug}", utils.Slugify(author))
	result = strings.ReplaceAll(result, "{ext}", ext)

	// Ensure sanitized
	result = utils.SanitizeFilename(result)

	// Slugify Filename?
	shouldSlugify := false
	if v, ok := metaOut["slugify-filename"]; ok {
		if b, ok := v.(bool); ok {
			shouldSlugify = b
		}
	} else if cfg.SlugifyFilename != nil {
		shouldSlugify = *cfg.SlugifyFilename
	}

	if shouldSlugify {
		ext := filepath.Ext(result)
		base := strings.TrimSuffix(result, ext)
		result = utils.Slugify(base) + ext
	}

	return result
}

// GetArgs converts a metadata map to pandoc arguments.
//
// Parameters:
//   - `meta`: the map of configuration options
//
// Returns:
//   - []string: a slice of command line arguments for pandoc
func GetArgs(meta map[string]interface{}) []string {
	var args []string

	// Check if `pandoc_args` exists and handle it separately
	var pandocArgs []string
	if val, ok := meta["pandoc_args"]; ok {
		if list, ok := val.([]interface{}); ok {
			for _, item := range list {
				pandocArgs = append(pandocArgs, fmt.Sprintf("%v", item))
			}
		}
		delete(meta, "pandoc_args")
	}

	// Sort keys for deterministic output
	var keys []string
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := meta[key]
		if key == "to" || key == "t" || key == "output" || key == "from" {
			continue
		}

		optName := key
		if strings.Contains(key, "_") {
			// Heuristic: check if we should replace _ with -
			// Ideally we check against pandoc supported options, but for now let's assume -
			optName = strings.ReplaceAll(key, "_", "-")
		}

		// Check if it's an internal flag to be ignored
		flagToCheck := "-" + optName
		if len(optName) > 1 {
			flagToCheck = "--" + optName
		}
		if internalFlags[flagToCheck] {
			continue
		}

		if val == false {
			continue
		}

		flag := "--" + optName

		switch v := val.(type) {
		case bool:
			if v {
				args = append(args, flag)
			}
		case []interface{}:
			for _, item := range v {
				args = append(args, flag, fmt.Sprintf("%v", item))
			}
		case map[string]interface{}:
			for k, subVal := range v {
				args = append(args, flag, fmt.Sprintf("%s=%v", k, subVal))
			}
		default:
			args = append(args, flag, fmt.Sprintf("%v", v))
		}
	}

	args = append(args, pandocArgs...)
	return args
}
