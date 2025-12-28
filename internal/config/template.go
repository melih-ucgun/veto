package config

import (
	"bytes"
	"text/template"
)

// ExecuteTemplate, verilen içeriği (content) sağlanan değişkenler (vars) ile işler.
func ExecuteTemplate(content string, vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("monarch").Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}
