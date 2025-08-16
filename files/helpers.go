package filestore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
)

func FileFromLocalPath(path string) (File, error) {
	stream, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	metadata := FileMetadata{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        info.Size(),
		ContentType: mime.TypeByExtension(filepath.Ext(path)),
	}
	return &file{metadata: metadata, stream: stream}, nil
}

func GenerateFileID(content []byte, originalName string) FileID {
	data := fmt.Sprintf("%s:%s", hex.EncodeToString(content), originalName)
	h := sha256.New()
	h.Write([]byte(data))
	hash := hex.EncodeToString(h.Sum(nil))
	return FileID(hash[:FileIDLength])
}

func GenerateFileIDFromStream(stream io.Reader, originalName string) (FileID, error) {
	h := sha256.New()
	_, err := io.Copy(h, stream)
	if err != nil {
		return InvalidFileID, err
	}
	contentHash := hex.EncodeToString(h.Sum(nil))
	data := fmt.Sprintf("%s:%s", contentHash, originalName)
	h.Reset()
	h.Write([]byte(data))
	finalHash := hex.EncodeToString(h.Sum(nil))
	return FileID(finalHash[:FileIDLength]), nil
}

func ExtractOriginalFileName(fileID FileID) string { return "" }
