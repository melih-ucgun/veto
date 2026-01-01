package core

import (
	"bytes"
	"text/template"
)

// ExecuteTemplate, verilen içeriği (content) sağlanan veri (data) ile işler.
// data genellikle *core.SystemContext olacaktır.
func ExecuteTemplate(content string, data interface{}) (string, error) {
	// "MissingKeyError" ile, olmayan bir değişken kullanılırsa hata vermesini sağlıyoruz.
	tmpl, err := template.New("veto").Option("missingkey=error").Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
