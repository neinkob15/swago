package swaggergen

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"github.com/go-chi/chi/v5"
)

type DocRouter struct {
	SwaggerVersion string `json:"openapi"`
	Info Info `json:"info"`
	Servers []Server `json:"servers"`
	Tags []Tag `json:"tags"`
	Paths Paths `json:"paths"`
	Components Components `json:"components"`
}
type Tag struct {
	Name string `json:"name"`
	Description string `json:"description"`
}
type Server struct {
	Url string `json:"url"`
	Description string `json:"description"`
}
type Components struct {
	Schemas Definitions `json:"schemas"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes"`
}
type SecurityScheme struct {
	Type string `json:"type"`
	Scheme string `json:"scheme"`
	BearerFormat string `json:"bearerFormat"`
}
type Definitions map[string]Definition
type Definition struct {
	Required []string `json:"required,omitempty"`
	Properties map[string]Property `json:"properties"`
}
type Property struct {
	Type string `json:"type"`
	Description string `json:"description,omitempty"`
	ReadOnly bool `json:"readOnly,omitempty"`
	WriteOnly bool `json:"writeOnly,omitempty"`
}
type Info struct {
	Version string `json:"version"`
	Title string `json:"title"`
	Description string `json:"description"`
}
type Paths map[string]Methods
type Methods map[string]FuncInfo

var authActivated bool
var definitions map[string]reflect.Value
var responseStates map[string]string
var tags map[string]string
var servers map[string]string
func init() {
	definitions = map[string]reflect.Value{}
	responseStates = map[string]string{}
	tags = map[string]string{}
	servers = map[string]string{}
}

func PrintRoutes(r chi.Routes) {
	var printRoutes func(parentPattern string, r chi.Routes)
	printRoutes = func(parentPattern string, r chi.Routes) {
		rts := r.Routes()
		for _, rt := range rts {
			if rt.SubRoutes == nil {
				fmt.Println(parentPattern + rt.Pattern)
			} else {
				pat := rt.Pattern

				subRoutes := rt.SubRoutes
				printRoutes(parentPattern+pat, subRoutes)
			}
		}
	}
	printRoutes("", r)
}

func RegisterType(name string, responseStatus int, val reflect.Value) {
	definitions[name] = val
	responseStates[name] = strconv.Itoa(responseStatus)
}
func RegisterDefaultResponse(name string, val reflect.Value) {
	definitions[name] = val
	responseStates[name] = "default"
}
func RegisterTag(name, description string) {
	tags[name] = description
}
func RegisterServer(name, description string) {
	servers[name] = description
}
func ActivateJWTAuth() {
	authActivated = true
}

func JSONRoutesDoc(r chi.Routes, title string, description string) string {
	doc, err := BuildDoc(r, title, description)
	if err != nil {
		panic(err)
	}
	v, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(v)
}
