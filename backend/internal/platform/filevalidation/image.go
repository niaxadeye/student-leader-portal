// Package filevalidation validates uploaded files without trusting multipart metadata.
package filevalidation

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

const sniffBytes = 512

var ErrInvalidImage = errors.New("invalid image content")

// InspectImage verifies that the declared MIME type, extension, and binary
// signature all describe the same supported image. The returned reader yields
// the complete original stream, including the bytes consumed for inspection.
func InspectImage(reader io.Reader, declaredType, originalName string) (io.Reader, string, error) {
	if reader == nil {
		return nil, "", ErrInvalidImage
	}

	header := make([]byte, sniffBytes)
	n, err := io.ReadFull(reader, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", err
	}
	header = header[:n]

	actualType := imageMIME(header)
	declaredType = strings.ToLower(strings.TrimSpace(strings.Split(declaredType, ";")[0]))
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(originalName), "."))
	if actualType == "" || declaredType != actualType || !extensionMatches(actualType, extension) {
		return nil, "", ErrInvalidImage
	}

	return io.MultiReader(bytes.NewReader(header), reader), actualType, nil
}

func imageMIME(header []byte) string {
	switch {
	case len(header) >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff:
		return "image/jpeg"
	case len(header) >= 8 && bytes.Equal(header[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return "image/png"
	case len(header) >= 6 && (bytes.Equal(header[:6], []byte("GIF87a")) || bytes.Equal(header[:6], []byte("GIF89a"))):
		return "image/gif"
	case len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return ""
	}
}

func extensionMatches(mime, extension string) bool {
	switch mime {
	case "image/jpeg":
		return extension == "jpg" || extension == "jpeg"
	case "image/png":
		return extension == "png"
	case "image/gif":
		return extension == "gif"
	case "image/webp":
		return extension == "webp"
	default:
		return false
	}
}
