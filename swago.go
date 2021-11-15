package swago

import (
	"fmt"
	"reflect"
	"strconv"
	"encoding/json"
	"bytes"
	"strings"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
	"net/http"
	"runtime"
	"go/parser"
	"go/token"
	"go/ast"
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
var helperFuncs []MethodWithFunc
var helperFuncName string

type MethodWithFunc struct{
	Method string
	Func func(w http.ResponseWriter, r *http.Request) (interface{}, error)
}

func init() {
	definitions = map[string]reflect.Value{}
	responseStates = map[string]string{}
	tags = map[string]string{}
	servers = map[string]string{}
	helperFuncs = []MethodWithFunc{}
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
func RegisterHelper(f func(w http.ResponseWriter, r *http.Request) (interface{}, error)) {
	_, file, no, ok := runtime.Caller(2)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var method string
	ast.Inspect(node, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if ok {
			if fset.Position(c.Pos()).Line == no {
				s, ok := c.Fun.(*ast.SelectorExpr)
				if ok {
					id := s.Sel
					if id.Name != "With" {
						method = strings.ToLower(id.Name)
						return false
					}
				}
			}
		}
		return true
	})
	if method == "" {
		panic("method could not be retrieved")
	}
	pc, _, _, ok := runtime.Caller(1)
    details := runtime.FuncForPC(pc)
    if ok && details != nil {
        helperFuncName = details.Name()
    } else {
		panic("there was a problem getting the name of the helper-function")
	}
	helperFuncs = append(helperFuncs, MethodWithFunc{
		Func: f,
		Method: method,
	})
}
func ActivateJWTAuth() {
	authActivated = true
}

func SwaggerRoutesDoc(r chi.Routes, title string, description string) string {
	doc, err := BuildDoc(r, title, description)
	if err != nil {
		panic(err)
	}
	v, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	j2, err := JSONToYAML(v)
	if err != nil {
		panic(err)
	}
	return string(j2)
}

func JSONToYAML(jsonData []byte) ([]byte, error) {
	return NewMarshaler().JSONToYAML(jsonData)
}
func NewMarshaler(os ...MarshalOption) *Marshaler {
	opts := &marshalOptions{
		Intend: 4, // Default for yaml.v3 package.
	}
	for _, o := range os {
		o(opts)
	}
	m := &Marshaler{}
	m.enc = yaml.NewEncoder(&m.buf)
	m.enc.SetIndent(opts.Intend)
	return m
}

type Marshaler struct {
	buf bytes.Buffer
	enc *yaml.Encoder
}

func (m *Marshaler) JSONToYAML(jsonData []byte) ([]byte, error) {
	n := &yaml.Node{}
	err := yaml.Unmarshal(jsonData, n)
	if err != nil {
		return nil, err
	}
	jsonToYAMLFormat(n)

	m.buf.Reset()
	err = m.enc.Encode(n)
	if err != nil {
		return nil, fmt.Errorf("marshal formated: %w", err)
	}
	return m.buf.Bytes(), nil
}

type MarshalOption func(opts *marshalOptions)
type marshalOptions struct {
	Intend int
}

func Indent(n int) MarshalOption {  return func(opts *marshalOptions) {opts.Intend = n } }

func jsonToYAMLFormat(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.SequenceNode, yaml.MappingNode:
		n.Style = yaml.LiteralStyle
	case yaml.ScalarNode:
		if n.Style == yaml.DoubleQuotedStyle {
			n.Style = yaml.FlowStyle
			if strings.Contains(n.Value, "\n") {
				n.Style = yaml.LiteralStyle
			}
		}
	}
	for _, c := range n.Content {
		jsonToYAMLFormat(c)
	}
}
