package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/athened/reborn-plugin-autoinstaller/config"
)

// PluginInfo describes a discovered plugin in the game's plugins folder.
type PluginInfo struct {
	Name        string // folder name suffix, e.g. "Ascy"
	LangCode    string // language code from .dat filename, e.g. "e", "ru"
	LangName    string // human-readable language name, e.g. "English"
	DatFile     string // filename, e.g. "SystemMsg-e.dat"
	DisplayName string // e.g. "[English] Ascy's custom chat"
}

var (
	pluginDirPattern = regexp.MustCompile(`^custom_systemMsg_(.+)$`)
	datFilePattern   = regexp.MustCompile(`^SystemMsg-(.+)\.dat$`)

	langNames = map[string]string{
		"e":  "English",
		"cn": "Chinese",
		"k":  "Korean",
		"ru": "Russian",
	}
)

// SourcePath returns the full path to the plugin's .dat file.
func SourcePath(gameDir, pluginName, langCode string) string {
	return filepath.Join(gameDir, "plugins",
		"custom_systemMsg_"+pluginName,
		"SystemMsg-"+langCode+".dat")
}

// DestPath returns the full path where the file should be installed.
func DestPath(gameDir, langCode string) string {
	return filepath.Join(gameDir, "system", "lang", langCode,
		"SystemMsg-"+langCode+".dat")
}

// ValidateGameDir checks that the expected sentinel file exists.
func ValidateGameDir(gameDir string) bool {
	// system/lang/e/SystemMsg-e.dat is present in all valid installs
	check := filepath.Join(gameDir, "system", "lang", "e", "SystemMsg-e.dat")
	_, err := os.Stat(check)
	return err == nil
}

// ScanPlugins scans <gameDir>/plugins/ for custom_systemMsg_<Name> directories.
// Each directory may contain one or more SystemMsg-<lang>.dat files.
// Returns the list of plugins, whether any had bad structure, and any error.
func ScanPlugins(gameDir string) (plugins []PluginInfo, badStructure bool, err error) {
	pluginsDir := filepath.Join(gameDir, "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := pluginDirPattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		name := m[1]
		dirPath := filepath.Join(pluginsDir, entry.Name())

		// Find all SystemMsg-<lang>.dat files in this plugin directory
		subEntries, readErr := os.ReadDir(dirPath)
		if readErr != nil {
			continue
		}

		foundAny := false
		for _, sub := range subEntries {
			if sub.IsDir() {
				continue
			}
			dm := datFilePattern.FindStringSubmatch(sub.Name())
			if dm == nil {
				continue
			}
			langCode := dm[1]
			langName, ok := langNames[langCode]
			if !ok {
				langName = strings.ToUpper(langCode)
			}
			displayName := fmt.Sprintf("[%s] %s's custom chat", langName, name)
			plugins = append(plugins, PluginInfo{
				Name:        name,
				LangCode:    langCode,
				LangName:    langName,
				DatFile:     sub.Name(),
				DisplayName: displayName,
			})
			foundAny = true
		}

		if !foundAny {
			// Folder matches pattern but contains no recognised .dat files
			badStructure = true
		}
	}
	return plugins, badStructure, nil
}

// Install copies the plugin file to its destination.
// Retries once on EBUSY (game client locking the file).
func Install(cfg *config.Config) error {
	src := SourcePath(cfg.GameDir, cfg.PluginName, cfg.PluginLang)
	dst := DestPath(cfg.GameDir, cfg.PluginLang)

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	err := copyFile(src, dst)
	if err != nil && isEBusy(err) {
		time.Sleep(300 * time.Millisecond)
		err = copyFile(src, dst)
	}
	return err
}

// DisplayName returns the user-friendly name for a plugin.
func DisplayName(pluginName, langCode string) string {
	if pluginName == "" {
		return ""
	}
	langName, ok := langNames[langCode]
	if !ok {
		langName = strings.ToUpper(langCode)
	}
	return fmt.Sprintf("[%s] %s's custom chat", langName, pluginName)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return out.Sync()
}

// HashFile returns the hex-encoded SHA-256 hash of the file at path.
// Returns an empty string and an error if the file cannot be read.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DestMatchesSource returns true when the destination file already has the
// same SHA-256 hash as the source plugin file — meaning no install is needed.
// If either file cannot be read, false is returned so we fall back to installing.
func DestMatchesSource(cfg *config.Config) (bool, string, string) {
	src := SourcePath(cfg.GameDir, cfg.PluginName, cfg.PluginLang)
	dst := DestPath(cfg.GameDir, cfg.PluginLang)

	srcHash, err := HashFile(src)
	if err != nil {
		return false, "", ""
	}
	dstHash, err := HashFile(dst)
	if err != nil {
		return false, srcHash, ""
	}
	return srcHash == dstHash, srcHash, dstHash
}

func isEBusy(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "used by another process") ||
		strings.Contains(s, "access is denied")
}
