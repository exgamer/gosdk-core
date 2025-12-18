package filefinder

import (
	enums2 "gitlab.almanit.kz/jmart/gosdk/pkg/localizer/enums"
	"os"
	"path/filepath"
)

func NewFileFinder() (*FileFinder, error) {
	wd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	wd += enums2.DirectoryLocalization

	f := &FileFinder{wd}

	return f, nil
}

type FileFinder struct {
	baseDir string
}

func (f *FileFinder) GetJsonFiles() ([]string, error) {
	// получаем путь к директории локализаций
	dirPath := filepath.Join(f.baseDir)

	// ищем все json файлы в директории
	var jsonFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".json" {
			jsonFiles = append(jsonFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return jsonFiles, nil
}
