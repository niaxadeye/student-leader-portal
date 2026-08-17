package filevalidation

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestInspectImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mime string
		file string
		data []byte
	}{
		{name: "jpeg", mime: "image/jpeg", file: "photo.JPG", data: []byte{0xff, 0xd8, 0xff, 0xe0, 1, 2}},
		{name: "png", mime: "image/png", file: "image.png", data: append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("payload")...)},
		{name: "gif", mime: "image/gif", file: "image.gif", data: []byte("GIF89apayload")},
		{name: "webp", mime: "image/webp", file: "image.webp", data: []byte("RIFF1234WEBPpayload")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			reader, mime, err := InspectImage(bytes.NewReader(test.data), test.mime+"; charset=binary", test.file)
			if err != nil || mime != test.mime {
				t.Fatalf("InspectImage() mime=%q err=%v", mime, err)
			}
			got, err := io.ReadAll(reader)
			if err != nil || !bytes.Equal(got, test.data) {
				t.Fatalf("returned stream=%x err=%v, want %x", got, err, test.data)
			}
		})
	}
}

func TestInspectImageRejectsSpoofedMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mime string
		file string
		data []byte
	}{
		{name: "non image", mime: "image/png", file: "image.png", data: []byte("not-an-image")},
		{name: "mime mismatch", mime: "image/jpeg", file: "image.png", data: append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0)},
		{name: "extension mismatch", mime: "image/png", file: "image.jpg", data: append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0)},
		{name: "missing reader", mime: "image/png", file: "image.png"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var reader io.Reader
			if test.data != nil {
				reader = bytes.NewReader(test.data)
			}
			_, _, err := InspectImage(reader, test.mime, test.file)
			if !errors.Is(err, ErrInvalidImage) {
				t.Fatalf("InspectImage() err=%v, want ErrInvalidImage", err)
			}
		})
	}
}
