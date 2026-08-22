package contests

import "testing"

func TestAvatarObjectKey(t *testing.T) {
	t.Parallel()
	key := avatarObjectKey("user-1", ImageUpload{KeySuffix: "abc", OriginalName: "Фото 1.PNG"})
	want := "avatars/user-1/abc-_____1.PNG"
	if key != want {
		t.Fatalf("got %q want %q", key, want)
	}
}

func TestSafeFileNameRejectsPath(t *testing.T) {
	t.Parallel()
	got := safeFileName(`..\..\secret.jpg`)
	if got != "secret.jpg" {
		t.Fatalf("got %q", got)
	}
}
