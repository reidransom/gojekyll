package site

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func extractRemoteThemeArchive(archivePath, destination string, limits remoteThemeLimits) (err error) {
	defer func() {
		if err != nil {
			_ = os.RemoveAll(destination)
		}
	}()

	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("open compressed archive: %w", err)
	}
	defer gzipReader.Close()

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	reader := tar.NewReader(gzipReader)
	var topLevel string
	var extracted int64
	var entries int64
	for {
		header, readErr := reader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read archive: %w", readErr)
		}
		entries++
		if entries > limits.Entries {
			return fmt.Errorf("archive exceeds %d entries", limits.Entries)
		}

		name, err := validArchivePath(header.Name)
		if err != nil {
			return err
		}
		components := strings.Split(name, "/")
		if topLevel == "" {
			topLevel = components[0]
		}
		if components[0] != topLevel {
			return fmt.Errorf("archive entries must share a top-level directory")
		}
		if len(components) == 1 {
			if header.Typeflag != tar.TypeDir {
				return fmt.Errorf("archive top-level entry must be a directory")
			}
			continue
		}

		relative := strings.Join(components[1:], "/")
		destinationPath := filepath.Join(destination, filepath.FromSlash(relative))
		if !pathWithin(destination, destinationPath) {
			return fmt.Errorf("archive entry %q escapes destination", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destinationPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > limits.FileBytes {
				return fmt.Errorf("archive file %q exceeds %d bytes", header.Name, limits.FileBytes)
			}
			extracted += header.Size
			if extracted > limits.ExtractedBytes {
				return fmt.Errorf("archive exceeds %d extracted bytes", limits.ExtractedBytes)
			}
			if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
				return err
			}
			if err := extractRegularFile(reader, destinationPath, header.Size); err != nil {
				return fmt.Errorf("extract %q: %w", header.Name, err)
			}
		default:
			return fmt.Errorf("archive entry %q has unsupported type", header.Name)
		}
	}

	if topLevel == "" {
		return fmt.Errorf("archive contains no entries")
	}
	layout := filepath.Join(destination, "_layouts", "default.html")
	info, err := os.Stat(layout)
	if err != nil {
		return fmt.Errorf("remote theme is missing _layouts/default.html: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("remote theme _layouts/default.html is not a regular file")
	}
	return nil
}

func validArchivePath(name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) || isWindowsAbsolutePath(name) {
		return "", fmt.Errorf("archive entry %q has an unsafe path", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("archive entry %q has an unsafe path", name)
	}
	return clean, nil
}

func isWindowsAbsolutePath(name string) bool {
	return len(name) >= 2 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':'
}

func pathWithin(root, filename string) bool {
	relative, err := filepath.Rel(root, filename)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func extractRegularFile(source io.Reader, destination string, size int64) (err error) {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	written, err := io.Copy(file, io.LimitReader(source, size))
	if err != nil {
		return err
	}
	if written != size {
		return io.ErrUnexpectedEOF
	}
	return nil
}
