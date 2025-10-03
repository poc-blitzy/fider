package tpl

import (
	"context"
	"html/template"
	"io"
	"path"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/i18n"
)

var cache = make(map[string]*template.Template)

func GetTemplate(baseFileName, templateFileName string) *template.Template {
	tmpl, ok := cache[templateFileName]
	if ok && !env.IsDevelopment() {
		return tmpl
	}

	baseFile := env.Path(baseFileName)
	templateFile := env.Path(templateFileName)
	tpl, err := template.New(path.Base(baseFile)).Funcs(templateFunctions).ParseFiles(baseFile, templateFile)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse template %s", templateFileName))
	}

	cache[templateFileName] = tpl
	return tpl
}

func Render(ctx context.Context, tmpl *template.Template, w io.Writer, data any) error {
	// Clone the template to avoid modifying the original
	cloned := template.Must(tmpl.Clone())
	
	// Copy the existing function map from templateFunctions to avoid modifying the global
	funcs := make(template.FuncMap)
	for k, v := range templateFunctions {
		funcs[k] = v
	}
	
	// Add the translate function with the current context
	funcs["translate"] = func(key string, params ...i18n.Params) string {
		return i18n.T(ctx, key, params...)
	}
	
	// Apply the merged function map and execute
	if err := cloned.Funcs(funcs).Execute(w, data); err != nil {
		return err
	}
	return nil
}
