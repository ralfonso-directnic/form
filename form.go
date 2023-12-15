package form

import (
	"errors"
	"fmt"
	"github.com/Masterminds/sprig"
	validator "github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	"github.com/jinzhu/copier"
	"github.com/nyaruka/phonenumbers"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var basetmpl string
var basepath string
var Funcs template.FuncMap

type Form struct {
	Tpl       *template.Template
	Decoder   *schema.Decoder
	Validator *validator.Validate
	selectMap map[string]map[string]interface{}
	Action    string
	Method    string
	Prefix    string
	Skip      []string
}

var ErrInvalidMethod = errors.New("Invalid Method")

func init() {

	basetmpl = `<div class="form-group">
    {{if ne .Type "hidden"}}
    <label class="form-label" {{with .ID}}for="{{.}}"{{end}}>
        {{.Label}}
    </label>
    {{end}}
    {{if eq .Type "textarea"}}
    <textarea {{.Attrs}} class="form-control" {{with .ID}}id="{{.}}"{{end}} name="{{.Name}}" rows="3" placeholder="{{.Placeholder}}">{{with .Value}}{{.}}{{end}}</textarea>
    {{else if eq .Type "checkbox" }}
    <input {{.Attrs}} type="{{.Type}}"  class="form-check-input" {{with .ID}}id="{{.}}"{{end}} name="{{.Name}}" placeholder="{{.Placeholder}}" {{with .Value}}value="{{.}}"{{end}}>
    {{else if eq .Type "select" }}
    <select {{.Attrs}} class="form-control" {{with .ID}}id="{{.}}"{{end}} name="{{.Name}}" {{.SelectType}}>
        {{ $myval := .Value }}
        {{ if gt (len .Placeholder) 0 }}
        <option value="" >{{ .Placeholder }}</option>
        {{ end }}
        {{ range $v,$k := .Items}}
          <option {{ if eq $myval $k  }}selected="selected"{{end}}value="{{$k}}">{{$v}}</option>
        {{end}}
    </select>
    {{ else }}
    <input {{.Attrs}} type="{{.Type}}" class="form-control" {{with .ID}}id="{{.}}"{{end}} name="{{.Name}}" placeholder="{{.Placeholder}}" {{with .Value}}value="{{.}}"{{end}}>
    {{end}}
    {{with .Footer}}
    <small class="form-text text-muted"> {{.}} </small>
    {{end}}
</div>`

}

//Load a base path for a template, optional
func BasePath(bp string) {

	basepath = bp

}

func New(pth ...string) (*Form, error) {
	var frmstr string
	var p string

	if len(pth) > 0 {

		p = pth[0]
	} else {
		p = ""
	}

	if len(basepath) > 0 {

		p = filepath.Join(basepath, p)

	}

	frm, errf := ioutil.ReadFile(p)
	if errf != nil {
		frmstr = basetmpl
	} else {

		frmstr = string(frm)
	}

	//embed our own funcs as well as sprig
	tpl := template.Must(template.New("form").Funcs(Funcs).Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"datetimelocal": func(val interface{}) (out string) {

			switch val.(type) {
			case string:
				out = val.(string)
			default:
				out = val.(time.Time).Format("2006-01-02T15:04")
			}

			return out

		},
		"phoneusa": func(val string) string {
			num, e := phonenumbers.Parse(val, "US")
			if e != nil {
				return val
			}
			return phonenumbers.Format(num, phonenumbers.NATIONAL)
		},
		"datetime": func(val interface{}) (out string) {

			switch val.(type) {
			case string:
				out = val.(string)
			default:
				out = val.(time.Time).Format("01/02/2006 15:04")
			}

			return out

		},
		"datelocal": func(val interface{}) (out string) {

			switch val.(type) {
			case string:
				out = val.(string)
			default:
				out = val.(time.Time).Format("01/02/2006")
			}

			return out
		},
		"date": func(val interface{}) (out string) {

			switch val.(type) {
			case string:
				out = val.(string)
			default:
				out = val.(time.Time).Format("2006-01-02")
			}

			return out
		},
	}).Parse(frmstr))

	decoder := schema.NewDecoder()
	vd := validator.New(validator.WithRequiredStructEnabled())

	return &Form{Tpl: tpl, Decoder: decoder, Validator: vd}, errf

}

func (f *Form) SkipField(skip string) {

	f.Skip = append(f.Skip, skip)
}

func (f *Form) Select(nm string, mp map[string]interface{}) {

	if f.selectMap == nil {
		f.selectMap = make(map[string]map[string]interface{})
	}

	f.selectMap[nm] = mp

}

///copy a source item to dest item and render, for example if you have a db result struct and a form struct, you can copy the db values to the form and then render it

func (f *Form) RenderBind(from interface{}, to interface{}, errs ...error) (template.HTML, error) {

	ce := copier.Copy(to, from)

	if ce != nil {

		errs = append(errs, ce)

	}

	return f.Render(to, errs...)

}

func (f *Form) DecodePost(req *http.Request, holder any) error {

	if req.Method == http.MethodPost && f.Decoder != nil {

		req.ParseForm()

		derr := f.Decoder.Decode(holder, req.PostForm)

		if derr != nil {

			return derr

		}

		return nil

	} else {

		return ErrInvalidMethod
	}

}

type ValidationError struct {
	Field string
	Value string
	Type  string
}

func (f *Form) Validate(holder any) (bool, []ValidationError) {

	var vee []ValidationError

	ve := f.Validator.Struct(holder)

	if ve != nil {

		if _, ok := ve.(*validator.InvalidValidationError); ok {

			return false, vee
		}

		for _, err := range ve.(validator.ValidationErrors) {

			//fmt.Println(err.Namespace())

			vee = append(vee, ValidationError{Field: err.Field(), Value: fmt.Sprint(err.Value()), Type: err.Tag()})

		}

		return false, vee
	}

	return true, vee
}

func (f *Form) RenderField(v interface{}, field_name string, errs ...error) (template.HTML, error) {

	fields := fields(v)
	errors := fieldErrors(errs)
	var html template.HTML
	for _, field := range fields {

		if field.Name != field_name {
			continue

		}
		field.Prefix = f.Prefix

		dump := false

		for _, sv := range f.Skip {

			if sv == field.Name {
				dump = true
				break
			}

			last := sv[len(sv)-1:]
			//nested struct, lets block anything with that dot
			if last == "." {

				if strings.Contains(field.Name, sv) {
					dump = true
					break
				}

			}

		}

		if dump == true {
			continue
		}

		if field.Type == "select" || field.Type == "checkbox" {

			if it, oks := f.selectMap[field.Name]; oks {

				field.Items = it

				//this block allows us to set the select value as an output ie CA=California, f.Value is CA and f.SelectValue is California
				for v, k := range it {
					if k == field.Value {
						field.SelectValue = v
					}

				}

			}

		}

		var sb strings.Builder

		f.Tpl.Funcs(template.FuncMap{
			"errors": func() []string {
				if errs, ok := errors[field.Name]; ok {
					return errs
				}
				return nil
			},
		})

		err := f.Tpl.Execute(&sb, field)
		if err != nil {
			return "", err
		}
		html = html + template.HTML(sb.String())
	}
	return html, nil

}

func (f *Form) Render(v interface{}, errs ...error) (template.HTML, error) {

	fields := fields(v)
	errors := fieldErrors(errs)
	var html template.HTML
	for _, field := range fields {

		field.Prefix = f.Prefix

		dump := false

		for _, sv := range f.Skip {

			if sv == field.Name {
				dump = true
				break
			}

			last := sv[len(sv)-1:]
			//nested struct, lets block anything with that dot
			if last == "." {

				if strings.Contains(field.Name, sv) {
					dump = true
					break
				}

			}

		}

		if dump == true {
			continue
		}

		if field.Type == "select" || field.Type == "checkbox" {

			if it, oks := f.selectMap[field.Name]; oks {

				field.Items = it

				//this block allows us to set the select value as an output ie CA=California, f.Value is CA and f.SelectValue is California
				for v, k := range it {
					if k == field.Value {
						field.SelectValue = v
					}

				}

			}

		}

		var sb strings.Builder

		f.Tpl.Funcs(template.FuncMap{
			"errors": func() []string {
				if errs, ok := errors[field.Name]; ok {
					return errs
				}
				return nil
			},
		})

		err := f.Tpl.Execute(&sb, field)
		if err != nil {
			return "", err
		}
		html = html + template.HTML(sb.String())
	}
	return html, nil

}
