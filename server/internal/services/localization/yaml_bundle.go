package localization

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAMLBundle reads configs/i18n/*.yaml into a Bundle.
// Each filename = locale; values are flat key→string maps.
// Returns nil bundle (with nil error) if directory doesn't exist — caller can fall back to MapBundle(nil).
func LoadYAMLBundle(dir string) (Bundle, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("i18n: read dir %s: %w", dir, err)
	}
	data := make(map[string]map[string]string)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		locale := e.Name()[:len(e.Name())-5]
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		m := map[string]string{}
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("i18n: parse %s: %w", e.Name(), err)
		}
		data[locale] = m
	}
	return MapBundle(data), nil
}
