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

const legacyTarRegularFile byte = 0 // tar.TypeRegA; emitted by codeload archives.

type remoteThemeArchiveState struct {
	topLevel  string
	extracted int64
	entries   int64
}

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
	state := remoteThemeArchiveState{}
	for {
		header, readErr := reader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read archive: %w", readErr)
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if err := state.extractEntry(reader, header, destination, limits); err != nil {
			return err
		}
	}

	if state.topLevel == "" {
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

func (state *remoteThemeArchiveState) extractEntry(reader *tar.Reader, header *tar.Header, destination string, limits remoteThemeLimits) error {
	state.entries++
	if state.entries > limits.Entries {
		return fmt.Errorf("archive exceeds %d entries", limits.Entries)
	}

	destinationPath, topLevel, err := state.entryDestination(header.Name, destination)
	if err != nil {
		return err
	}
	if topLevel {
		if !archiveEntryIsDirectory(header) {
			return fmt.Errorf("archive top-level entry must be a directory")
		}
		return nil
	}
	return state.extractArchiveEntry(reader, header, destinationPath, limits)
}

func (state *remoteThemeArchiveState) entryDestination(name, destination string) (string, bool, error) {
	name, err := validArchivePath(name)
	if err != nil {
		return "", false, err
	}
	components := strings.Split(name, "/")
	if state.topLevel == "" {
		state.topLevel = components[0]
	}
	if components[0] != state.topLevel {
		return "", false, fmt.Errorf("archive entries must share a top-level directory")
	}
	if len(components) == 1 {
		return "", true, nil
	}

	destinationPath := filepath.Join(destination, filepath.FromSlash(strings.Join(components[1:], "/")))
	if !pathWithin(destination, destinationPath) {
		return "", false, fmt.Errorf("archive entry %q escapes destination", name)
	}
	return destinationPath, false, nil
}

func (state *remoteThemeArchiveState) extractArchiveEntry(reader *tar.Reader, header *tar.Header, destination string, limits remoteThemeLimits) error {
	switch {
	case archiveEntryIsDirectory(header):
		return os.MkdirAll(destination, 0o755)
	case isArchiveRegularFile(header):
		return state.extractArchiveRegularFile(reader, header, destination, limits)
	default:
		return fmt.Errorf("archive entry %q has unsupported type", header.Name)
	}
}

func isArchiveRegularFile(header *tar.Header) bool {
	return header.Typeflag == tar.TypeReg || header.Typeflag == legacyTarRegularFile
}

func (state *remoteThemeArchiveState) extractArchiveRegularFile(reader *tar.Reader, header *tar.Header, destination string, limits remoteThemeLimits) error {
	if header.Size < 0 || header.Size > limits.FileBytes {
		return fmt.Errorf("archive file %q exceeds %d bytes", header.Name, limits.FileBytes)
	}
	state.extracted += header.Size
	if state.extracted > limits.ExtractedBytes {
		return fmt.Errorf("archive exceeds %d extracted bytes", limits.ExtractedBytes)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := extractRegularFile(reader, destination, header.Size); err != nil {
		return fmt.Errorf("extract %q: %w", header.Name, err)
	}
	return nil
}

func archiveEntryIsDirectory(header *tar.Header) bool {
	return header.Typeflag == tar.TypeDir || header.FileInfo().IsDir()
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
