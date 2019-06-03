package main

import (
	"html/template"
	"io/ioutil"
	"strings"
)

var TemplateRoot = template.New("")

func init() {
	templateDir, err := assets.Open("/template/")
	if err != nil {
		panic(err)
	}
	dir, ok := templateDir.(*vfsgen۰Dir)
	if !ok {
		panic("/template/ was not a directory")
	}
	files, err := dir.Readdir(0)
	if err != nil {
		panic("error reading /template/: " + err.Error())
	}
	for _, templateFile := range files {
		if !strings.HasSuffix(templateFile.Name(), ".tmpl") {
			continue
		}
		templateName := strings.TrimSuffix(templateFile.Name(), ".tmpl")
		tmplFile, err := assets.Open("template/" + templateFile.Name())
		if err != nil {
			panic("opening template file " + templateFile.Name() + ": " + err.Error())
		}
		defer tmplFile.Close()
		tmplBytes, err := ioutil.ReadAll(tmplFile)
		if err != nil {
			panic("reading template file " + templateFile.Name() + ": " + err.Error())
		}
		template.Must(TemplateRoot.New(templateName).Parse(string(tmplBytes)))
	}
}
