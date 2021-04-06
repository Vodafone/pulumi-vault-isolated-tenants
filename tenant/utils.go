package tenant

import (
	"bytes"
	"html/template"
)

// Check is a poor man's error handling
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// TemplateConfig is a Configuration structure for templates in this repository
type TemplateConfig struct {
	Tenant PrefixArgs
}

func readTemplate(templatePath string, config TemplateConfig) string {
	policyTemplate, err := template.ParseFiles(templatePath)
	Check(err)

	var tpl bytes.Buffer
	err = policyTemplate.Execute(&tpl, config)
	Check(err)

	return tpl.String()
}
