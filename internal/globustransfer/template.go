package globustransfer

import (
	"bytes"
	"strings"
	"text/template"
)

var datasetDestPathTemplate *template.Template = template.New("dataset destination path template").Funcs(
	template.FuncMap{
		"replace": func(s string, query string, repl string) string {
			return strings.ReplaceAll(s, query, repl)
		},
	},
)

type destPathParamsStruct struct {
	DatasetFolder string
	SourceFolder  string
	Pid           string
	PidShort      string
	PidPrefix     string
	PidEncoded    string
	Username      string
}

func templateDestinationFolder(data destPathParamsStruct) (string, error) {
	buffer := bytes.Buffer{}
	err := datasetDestPathTemplate.Execute(&buffer, data)
	return buffer.String(), err
}

func SetTemplateForDestinationPath(template string) error {
	var err error
	datasetDestPathTemplate, err = datasetDestPathTemplate.Parse(template)
	return err
}
