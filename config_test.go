package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/adrg/xdg"
)

func prepareConfigPath() (func(), error) {
	cfgPath, err := os.MkdirTemp("", "ccienv-config-test-")
	if err != nil {
		return func() {}, err
	}
	os.Setenv("XDG_CONFIG_HOME", cfgPath)
	xdg.Reload()
	return func() { _ = os.RemoveAll(cfgPath) }, nil
}

func prepareConfig(data string) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	return os.WriteFile(path, []byte(data), 0644)
}

func TestReadConfig(t *testing.T) {
	cleaner, err := prepareConfigPath()
	if err != nil {
		t.Errorf("Failed to prepare: %v", err)
	}
	defer cleaner()
	if err := prepareConfig("apitoken: abc\norganizationname: cde"); err != nil {
		t.Errorf("Failed to prepare a config file: %v", err)
	}
	got, err := ReadConfig()
	if err != nil {
		t.Error(err)
	}
	expected := &Config{
		ApiToken:         "abc",
		OrganizationName: "cde",
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ReadConfig() got = %v, want %v", got, expected)
	}
}
