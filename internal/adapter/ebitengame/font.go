package ebitengame

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

type Faces struct {
	Small  font.Face
	Normal font.Face
	Large  font.Face
}

func loadFaces() Faces {
	fontBytes, err := loadFontBytes()
	if err != nil {
		log.Fatalf("не удалось загрузить файл шрифта CyberMono.ttf: %v", err)
	}

	fnt, err := truetype.Parse(fontBytes)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	mk := func(size float64) font.Face {
		return truetype.NewFace(fnt, &truetype.Options{Size: size, DPI: 72})
	}
	return Faces{
		Small:  mk(14),
		Normal: mk(18),
		Large:  mk(26),
	}
}

func loadFontBytes() ([]byte, error) {
	searchPaths := []string{
		filepath.Join("internal", "adapter", "ebitengame", "assets", "CyberMono.ttf"),
		filepath.Join("assets", "PressStart2P-Regular.ttf"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		searchPaths = append([]string{
			filepath.Join(exeDir, "assets", "PressStart2P-Regular.ttf"),
			filepath.Join(exeDir, "PressStart2P-Regular.ttf"),
		}, searchPaths...)
	}

	var lastErr error
	for _, path := range searchPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("файл шрифта CyberMono.ttf не найден; разместите его рядом с бинарником в папке assets или в internal/adapter/ebitengame/assets (последняя ошибка: %v)", lastErr)
}
