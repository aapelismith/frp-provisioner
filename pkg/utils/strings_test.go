package utils_test

import (
	"bytes"
	"github.com/google/uuid"
	"io"
	"kunstack.com/pharos/pkg/utils"
	"os"
	"path/filepath"
	"testing"
)

func TestFileOrContent(t *testing.T) {
	a := filepath.Join(os.TempDir(), uuid.New().String())

	result := utils.FileOrContent(a)

	if result != a {
		t.Fatalf("expected %s, got: %s", a, result)
	}

	payload := []byte(uuid.New().String())

	data := bytes.NewBuffer(payload)

	f, err := os.OpenFile(a, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.Remove(a)
	}()

	if _, err := io.Copy(f, data); err != nil {
		t.Fatal(err)
	}

	_ = f.Close()

	result = utils.FileOrContent(a)

	if result != string(payload) {
		t.Fatalf("expected %s, got: %s", payload, result)
	}
}
