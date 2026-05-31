package importer

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
	"time"
)

const defaultDestTemplate = "{{ .Owner }}'s Library/{{ .Year }}/{{ .Month }}/{{ .Day }}"

type destVars struct {
	Owner, Year, Month, Day, CameraModel, CardUUID string
}

func resolveDestDir(importRoot string, tmpl *template.Template, owner, cardUUID, cameraModel string, dt time.Time) (string, error) {
	vars := destVars{
		Owner:       owner,
		CardUUID:    cardUUID,
		CameraModel: cameraModel,
		Year:        dt.Format("2006"),
		Month:       dt.Format("01"),
		Day:         dt.Format("02"),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("exec dest template: %w", err)
	}
	return filepath.Join(importRoot, filepath.FromSlash(buf.String())), nil
}
