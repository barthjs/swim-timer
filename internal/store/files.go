package store

import "strings"

// maxFileNameBytes is the longest file name common file systems accept, counted in bytes.
const maxFileNameBytes = 255

// fileName returns the name of the item kept in the file at a storage path, and false for a file the store
// did not write, such as one in a subfolder, one with another extension, or one whose name filePath rejects.
func fileName(folder, p, extension string) (string, bool) {
	base, found := strings.CutPrefix(p, folder+"/")
	if !found {
		return "", false
	}
	name, found := strings.CutSuffix(base, extension)
	if !found {
		return "", false
	}
	if _, err := filePath(folder, name, extension); err != nil {
		return "", false
	}
	return name, true
}

// filePath returns the storage path of the file that keeps the item with the given name in folder, or ErrInvalidName
// for a name that does not address a file directly inside the folder, since fileName only finds those.
// A name whose file name would exceed maxFileNameBytes is rejected as well, since the file could not be written.
func filePath(folder, name, extension string) (string, error) {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", ErrInvalidName
	}
	if len(name)+len(extension) > maxFileNameBytes {
		return "", ErrInvalidName
	}
	return folder + "/" + name + extension, nil
}
