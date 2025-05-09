# Form

Easily create HTML forms with Go structs.

## Overview

The `form` package makes it easy to take a Go struct and turn it into an HTML form using whatever HTML format you want. Below is an example, along with the output, but first let's just look at an example of what I mean.

Let's say you have a Go struct that looks like this:

```go
type customer struct {
	Name    string
	Email   string
	Address *address
}

type address struct {
	Street1 string
	Street2 string
	City    string
	State   string
	Zip     string `form:"label=Postal Code"`
}
```

Now you want to generate an HTML form for it, but that is somewhat annoying if you want to persist user-entered values if there is an error, or if you want to support loading URL query params and auto-filling the form for the user. With this package you can very easily do both of those things simply by defining what the HTML for an input field should be:

```html
<div class="mb-4">
	<label class="block text-grey-darker text-sm font-bold mb-2" {{with .ID}}for="{{.}}"{{end}}>
		{{.Label}}
	</label>
	<input class="shadow appearance-none border rounded w-full py-2 px-3 text-grey-darker leading-tight {{if errors}}border-red{{end}}" {{with .ID}}id="{{.}}"{{end}} type="{{.Type}}" name="{{.Name}}" placeholder="{{.Placeholder}}" {{with .Value}}value="{{.}}"{{end}}>
	{{range .Errors}}
		<p class="text-red pt-2 text-xs italic">{{.}}</p>
	{{end}}
</div>
```



## Alternative Syntax

```
    var u User{}
    var ufrm SignupForm{}
    
    //optionally provide a path to look at templates
    form.BasePath("/tmp/form")
    
    //Create the form
    
    frm, err := form.New("/formbasic.html")
    if(err!=nil){//handle error}
    frm.Render(ufrm)
    //- or -
    frm.RenderBind(u,ufrm) //this would copy the values into ufrm before rendering


```

## Handle and validate a post using the go-playground validator


```
    type Info struct {
        Name    string `form:"required" validate:"required"`
        Email   string `form:"required" validate:"email,required"`
        Street1 string `form:"required"  validate:"required"`
        Street2 string
        City    string
        State   string `validate:"len=2,alpha"`
        Zip     string `form:"label=Postal Code"`
    }
    
    //set a base path for form to look for
    form.BasePath("./tmpl")
    
    //load basic.html tempalte
	frm, err := form.New("/basic.html") 

	a := &Info{}

	//decode a post, returns err if no post
	derr := frm.DecodePost(req, a)

	if derr == nil {

		stat, errs := frm.Validate(a)

		if !stat {
			fmt.Println("Validation Error", errs)
			//Validation Error [{Email  email} {Street1  required}]

		}

	}


	var s string
	if err == nil { //handle error}
		out, _ := frm.Render(a, errs)
		s = fmt.Sprint(out)
	} else {
		s = fmt.Sprint(err)
	}



```

## Installation

To install this package, simply `go get` it:

```
go get github.com/joncalhoun/form
```

## Complete Examples

This entire example can be found in the [examples/readme](examples/readme) directory. Additional examples can also be found in the [examples/](examples/) directory and are a great way to see how this package could be used.

**Source Code**

```go
package main

import (
	"html/template"
	"net/http"

	"github.com/joncalhoun/form"
)

var inputTpl = `
<label {{with .ID}}for="{{.}}"{{end}}>
	{{.Label}}
</label>
<input {{with .ID}}id="{{.}}"{{end}} type="{{.Type}}" name="{{.Name}}" placeholder="{{.Placeholder}}" {{with .Value}}value="{{.}}"{{end}}>
{{with .Footer}}
  <p>{{.}}</p>
{{end}}
`

type Address struct {
	Street1 string `form:"label=Street;placeholder=123 Sample St"`
	Street2 string `form:"label=Street (cont);placeholder=Apt 123"`
	City    string
	State   string `form:"footer=Or your Province"`
	Zip     string `form:"label=Postal Code"`
	Country string
}

func main() {
	tpl := template.Must(template.New("").Parse(inputTpl))
	fb := form.Builder{
		InputTemplate: tpl,
	}

	pageTpl := template.Must(template.New("").Funcs(fb.FuncMap()).Parse(`
		<html>
		<body>
			<form>
				{{inputs_for .}}
			</form>
		</body>
		</html>`))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		pageTpl.Execute(w, Address{
			Street1: "123 Known St",
			Country: "United States",
		})
	})
	http.ListenAndServe(":3000", nil)
}
```

**Relevant HTML** trimmed for brevity


```html
<form>
  <label >
    Street
  </label>
  <input  type="text" name="Street1" placeholder="123 Sample St" value="123 Known St">

  <label >
    Street (cont)
  </label>
  <input  type="text" name="Street2" placeholder="Apt 123" >

  <label >
    City
  </label>
  <input  type="text" name="City" placeholder="City" >

  <label >
    State
  </label>
  <input  type="text" name="State" placeholder="State" >
  <p>Or your Province</p>

  <label >
    Postal Code
  </label>
  <input  type="text" name="Zip" placeholder="Postal Code" >

  <label >
    Country
  </label>
  <input  type="text" name="Country" placeholder="Country" value="United States">
</form>
```

## How it works

The `form.Builder` type provides a single method - `Inputs` - which will parse the provided struct to determine which fields it contains, any values set for each field, and any struct tags provided for the form package. Once that information is parsed it will execute the provided `InputTemplate` field in the builder for each field in the struct, **including nested fields**.

Most of the time you will probably want to just make this helper available to your html templates via the `template.Funcs()` functions and the `template.FuncMap` type, as I did in the example above.

## I don't recommend tagging domain types

It is also worth mentioning that I don't really recommend adding `form` struct tags to your domain types, and I typically create types specifically used to generate forms. Eg:

```go
// This is my domain type
type User struct {
  ID           int
  Name         string
  Email        string
  PasswordHash string
}

// Somewhere else I'll create my html-specific type:
type signupForm struct {
  Name         string `form:"..."`
  Email        string `form:"type=email"`
  Password     string `form:"type=password"`
  Confirmation string `form:"type=password;label=Password Confirmation"`
}
```

## Parsing submitted forms

If you also need to parse forms created by this package, I recommend using the [gorilla/schema](https://github.com/gorilla/schema) package. This package *should* generate input names compliant with the `gorilla/schema` package by default, so as long as you don't change the names it should be pretty trivial to decode.

There is an example of this in the [examples/tailwind](examples/tailwind) directory.

## Rendering errors

If you want to render errors, see the [examples/errors/errors.go](examples/errors/errors.go) example and most notably check out the `inputs_and_errors_for` function provided to templates via the `Builder.FuncMap()` function.

*TODO: Add some better examples here, but the provided code sample **is** a complete example.*

## This may have bugs

This is a very early iteration of the package, and while it appears to be working for my needs chances are it doesn't cover every use case. If you do find one that isn't covered, try to provide a PR with a breaking test.


## Notes

This section is mostly for myself to jot down notes, but feel free to read away.

### Potential features

#### Parsing forms

Long term this could also support parsing forms, but gorilla/schema does a great job of that already so I don't see any reason to at this time. It would likely be easier to just make the default input names line up with what gorilla/schema expects and provide examples for how to use the two together.

#### Checkboxes and other data types

Maybe allow for various templates for different types, but for now this is possible to do in the HTML templates so it isn't completely missing.

#### Headers on nested structs

Let's say we have this type:

```go
type Nested struct {
  Name string
  Email string
  Address Address
}

type Address struct {
  Street1 string
  Street2 string
  // ...
}
```

It might make sense to make an optional way to add headers in the form when the nested Address portion is rendered, so the form looks like:

```
Name:    [    ]
Email:   [    ]

<Address Header Here>

Street1: [    ]
Street2: [    ]
...
```

This *should* be pretty easy to do with struct tags on the `Address Address` line.
