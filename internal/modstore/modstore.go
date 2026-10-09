// Package modstore holds the durable-registry scaffolding the Ashveil
// modules share: loading a registry file through the plugin API and the
// "is persistence usable" gate every save checks first.
package modstore

import (
	"errors"
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/plugins"
)

// Load reads the plugin data file name into out. A missing file loads the
// empty registry. Any other file is decoded by decode here, because the
// plugin's ReadIntoStruct discards YAML errors, which would turn unreadable
// data into an empty, writable registry.
func Load[R any](plug *plugins.Plugin, name string, empty func() R, decode func([]byte, *R) error, out *R) error {
	data, err := plug.ReadBytes(name)
	if errors.Is(err, os.ErrNotExist) {
		*out = empty()
		return nil
	}
	if err != nil {
		return err
	}
	return decode(data, out)
}

// Save writes registry to the plugin data file name.
func Save[R any](plug *plugins.Plugin, name string, registry R) error {
	return plug.WriteStruct(name, registry)
}

// Available reports whether a module may save: its last load must have
// succeeded and a store must be wired. module prefixes the error text.
func Available(module string, loadErr error, storeSet bool) error {
	if loadErr != nil {
		return fmt.Errorf("%s: persistence unavailable until a successful reload: %w", module, loadErr)
	}
	if !storeSet {
		return fmt.Errorf("%s: persistence unavailable", module)
	}
	return nil
}
