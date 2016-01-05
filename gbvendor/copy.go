package vendor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb/fileutils"
)

const debugCopypath = true
const debugCopyfile = false

// Copypath copies the contents of src to dst, excluding any file or
// directory that starts with a period.
func Copypath(dst string, src string) error {
	var symlinks []string
	
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return os.MkdirAll(
				filepath.Join(dst, path[len(src):]),
				info.Mode() & os.ModePerm)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// postpone symlink creation to last
			symlinks = append(symlinks,path)
			return nil
		}

		dst := filepath.Join(dst, path[len(src):])
		return copyfile(dst, path)
	})

	// Handle symlinks last
	for _, path := range symlinks {
		target, err := os.Readlink(path)
		if err != nil {
			if debugCopypath {
				fmt.Printf("reading symlink error (%v): %s\n", path, err)
			}
			goto CLEANUP
		}
		dst := filepath.Join(dst, path[len(src):])
		fmt.Printf("making symlink: %v -> %v\n", dst, target)
		err = os.Symlink(target, dst)
		if err != nil {
			if debugCopypath {
				fmt.Printf("making symlink error (%v): %s\n", path, err)
			}
			goto CLEANUP
		}
	}

CLEANUP:
	if err != nil {
		// if there was an error during copying, remove the partial copy.
		fileutils.RemoveAll(dst)
	}
	return err
}

func copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	defer w.Close()
	if debugCopyfile {
		fmt.Printf("copyfile(dst: %v, src: %v)\n", dst, src)
	}
	_, err = io.Copy(w, r)
	if err != nil {
		fmt.Printf("copyfile(dst: %v, src: %v) %s\n", dst, src, err)
	}
	return err
}
