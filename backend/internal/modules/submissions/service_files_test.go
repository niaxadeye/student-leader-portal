package submissions

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestInspectImageUpload(t *testing.T) {
	t.Parallel()
	data := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("payload")...)
	in := UploadInput{
		OriginalName: "proof.png", ContentType: "image/png; charset=binary",
		Reader: bytes.NewReader(data),
	}
	prepared, err := inspectImageUpload(in, "png")
	if err != nil || prepared.ContentType != "image/png" {
		t.Fatalf("inspectImageUpload() type=%q err=%v", prepared.ContentType, err)
	}
	got, err := io.ReadAll(prepared.Reader)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("returned stream=%x err=%v, want %x", got, err, data)
	}
}

func TestInspectImageUploadRejectsSpoofedImage(t *testing.T) {
	t.Parallel()
	_, err := inspectImageUpload(UploadInput{
		OriginalName: "proof.png", ContentType: "image/png", Reader: bytes.NewReader([]byte("not an image")),
	}, "png")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("inspectImageUpload() err=%v, want ErrValidation", err)
	}
}

func TestInspectImageUploadLeavesOtherFormatsUntouched(t *testing.T) {
	t.Parallel()
	reader := bytes.NewReader([]byte("document"))
	prepared, err := inspectImageUpload(UploadInput{
		OriginalName: "brief.pdf", ContentType: "application/pdf", Reader: reader,
	}, "pdf")
	if err != nil || prepared.Reader != reader || prepared.ContentType != "application/pdf" {
		t.Fatalf("inspectImageUpload()=%#v err=%v", prepared, err)
	}
}
