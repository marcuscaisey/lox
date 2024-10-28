package metamodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
)

// Load downloads the meta model for the given LSP version from Microsoft's website and returns it.
// Once downloaded, the meta model is cached in the user's cache directory and will be loaded from there on subsequent
// calls.
func Load(version string) (*MetaModel, error) {
	data, err := readOrDownload(version)
	if err != nil {
		return nil, fmt.Errorf("loading meta model: %s", err)
	}

	var model *MetaModel
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields() // This should catch any updates to the model that we're not aware of.
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("loading meta model: unmarshaling from JSON: %s", err)
	}

	return model, nil
}

func readOrDownload(version string) ([]byte, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("checking cache directory: %w", err)
	}
	cachePath := fmt.Sprintf("%s/loxls/typegen/metamodels/%s.json", cacheDir, version)
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading from cache: %w", err)
	}

	url := fmt.Sprintf("https://microsoft.github.io/language-server-protocol/specifications/lsp/%s/metaModel/metaModel.json", version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading: non-200 response from %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloading: reading response body: %w", err)
	}
	if err := os.MkdirAll(path.Dir(cachePath), 0750); err != nil {
		return nil, fmt.Errorf("writing to cache: %w", err)
	}
	if err := os.WriteFile(cachePath, body, 0644); err != nil {
		return nil, fmt.Errorf("writing to cache: %w", err)
	}

	return body, nil
}
