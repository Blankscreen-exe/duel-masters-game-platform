// Package avatars holds the pictures players can choose for their profile. They
// are part of the server's program, so there is nothing to install beside it, and
// adding a file to images is all it takes to offer another.
package avatars

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

const imageExtension = ".png"

//go:embed images/*.png
var files embed.FS

// Image is one of the pictures.
type Image struct {
	// ID is what an account keeps to say which picture it has: the file's name
	// without its extension.
	ID string
	// Data is the PNG file.
	Data []byte
	// Version changes when the file does, for browsers to check their copy by.
	Version string
}

var (
	all  = mustRead()
	byID = func() map[string]Image {
		index := make(map[string]Image, len(all))
		for _, image := range all {
			index[image.ID] = image
		}
		return index
	}()
)

// mustRead reads the pictures, by name. Being part of the program, they can only
// fail to read if the program itself is broken.
func mustRead() []Image {
	entries, err := fs.ReadDir(files, "images")
	if err != nil {
		panic(fmt.Sprintf("reading the embedded avatars: %v", err))
	}

	images := make([]Image, 0, len(entries))
	for _, entry := range entries {
		data, err := files.ReadFile(path.Join("images", entry.Name()))
		if err != nil {
			panic(fmt.Sprintf("reading the embedded avatar %s: %v", entry.Name(), err))
		}
		sum := sha256.Sum256(data)
		images = append(images, Image{
			ID:      strings.TrimSuffix(entry.Name(), imageExtension),
			Data:    data,
			Version: hex.EncodeToString(sum[:8]),
		})
	}
	slices.SortFunc(images, func(a, b Image) int { return strings.Compare(a.ID, b.ID) })

	return images
}

// IDs lists the pictures players can choose, in the order to show them.
func IDs() []string {
	ids := make([]string, 0, len(all))
	for _, image := range all {
		ids = append(ids, image.ID)
	}
	return ids
}

// ByID returns a picture, and false if there is none with the id.
func ByID(id string) (Image, bool) {
	image, found := byID[id]
	return image, found
}
