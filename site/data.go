package site

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/reidransom/jigyll/utils"
)

func (s *Site) readDataFiles() error {
	dataDir := filepath.Join(s.SourceDir(), s.cfg.DataDir)
	data, err := readDataDir(dataDir)
	if err != nil {
		return err
	}
	s.data = data
	return nil
}

// readDataDir reads a data directory, recursing into subdirectories and
// namespacing their contents under the directory name, as Jekyll does
// (_data/orgs/jekyll.yml -> site.data.orgs.jekyll).
func readDataDir(dir string) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, err
	}
	for _, f := range files {
		filename := filepath.Join(dir, f.Name())
		if f.IsDir() {
			sub, err := readDataDir(filename)
			if err != nil {
				return nil, err
			}
			data[f.Name()] = sub
			continue
		}
		d, err := readDataFile(filename)
		if err != nil {
			return nil, utils.WrapPathError(err, filename)
		}
		if d != nil {
			data[utils.TrimExt(f.Name())] = d
		}
	}
	return data, nil
}

func readDataFile(filename string) (interface{}, error) {
	switch filepath.Ext(filename) {
	case ".csv":
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close() // nolint: errcheck
		r := csv.NewReader(f)
		return r.ReadAll()
	case ".json":
		b, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var d interface{}
		err = json.Unmarshal(b, &d)
		return d, err
	case ".yaml", ".yml":
		b, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var d interface{}
		err = utils.UnmarshalYAMLInterface(b, &d)
		return d, err
	}
	return nil, nil
}
