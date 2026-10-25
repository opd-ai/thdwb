package pages

import (
	"bytes"
	"html/template"
	"os"
	"strings"
)

type FileBrowser struct {
	Path       string
	Parent     string
	Dirs       []string
	Files      []string
	NotRootDir bool
}

func RenderFileBrowser(path string) string {
	renderedTemplate := new(bytes.Buffer)
	fileBrowser := &FileBrowser{Path: path}

	spltPath := strings.Split(path, "/")

	if len(spltPath) <= 2 {
		fileBrowser.Parent = "/"
		fileBrowser.NotRootDir = false
	} else {
		fileBrowser.Parent = strings.Join(spltPath[:len(spltPath)-1], "/")
		fileBrowser.NotRootDir = true
	}

	files, _ := os.ReadDir(path)
	for _, file := range files {
		if file.IsDir() {
			fileBrowser.Dirs = append(fileBrowser.Dirs, file.Name())
		} else {
			fileBrowser.Files = append(fileBrowser.Files, file.Name())
		}
	}

	t, err := template.New("fileBrowser").Parse(fileBrowserTemplate())
	if err != nil {
		return err.Error()
	}
	if err := t.ExecuteTemplate(renderedTemplate, "fileBrowser", fileBrowser); err != nil {
		return err.Error()
	}

	return renderedTemplate.String()
}
