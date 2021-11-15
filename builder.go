package swago

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
	"github.com/go-chi/chi/v5"
)

func BuildDoc(r chi.Routes, title, description string) (DocRouter, error) {
	dr := DocRouter{SwaggerVersion: "3.0.1",
		Info: Info{
			Version:     "1.0.0",
			Title:       title,
			Description: description,
		},
	}

	for k, v := range tags {
		dr.Tags = append(dr.Tags, Tag{
			Name:        k,
			Description: v,
		})
	}
	for k, v := range servers {
		dr.Servers = append(dr.Servers, Server{
			Url:         k,
			Description: v,
		})
	}

	// Walk and generate the router docs
	dr.Paths = buildDocRouterPaths(r, "")

	defs := map[string]Definition{}
	for key, v := range definitions {
		props := map[string]Property{}
		typeOfS := v.Type()
		var requiredProps []string
		for i := 0; i < v.NumField(); i++ {
			t := v.Field(i).Type().String()
			if t == "int" {
				t = "number"
			}
			t = strings.TrimPrefix(t, "*")
			tags, err := structtag.Parse(string(typeOfS.Field(i).Tag))
			if err != nil {
				return DocRouter{}, err
			}
			swagTag, _ := tags.Get("swago")
			jsonTag, err := tags.Get("json")
			tagName := typeOfS.Field(i).Name
			if err == nil {
				tagName = jsonTag.Name
			}
			readOnly := false
			writeOnly := false
			if swagTag != nil {
				for _, option := range swagTag.Options {
					if option == "readOnly" {
						readOnly = true
					} else if option == "writeOnly" {
						writeOnly = true
					} else if option == "required" {
						requiredProps = append(requiredProps, tagName)
					}
				}
				props[tagName] = Property{
					Type:        t,
					Description: swagTag.Name,
					WriteOnly:   writeOnly,
					ReadOnly:    readOnly,
				}
			} else {
				props[tagName] = Property{
					Type:        t,
					Description: "",
					WriteOnly:   writeOnly,
					ReadOnly:    readOnly,
				}
			}

		}
		defs[key] = Definition{
			Required:   requiredProps,
			Properties: props,
		}
	}
	var securitySchemes map[string]SecurityScheme
	if authActivated {
		securitySchemes = map[string]SecurityScheme{}
		securitySchemes["bearerAuth"] = SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"}
	}

	components := Components{
		Schemas:         defs,
		SecuritySchemes: securitySchemes,
	}
	dr.Components = components

	return dr, nil
}

func buildDocRouterPaths(r chi.Routes, prefix string) Paths {
	rts := r
	drts := Paths{}

	for _, rt := range rts.Routes() {
		drt := Methods{}
		if rt.SubRoutes != nil {
			subRoutes := rt.SubRoutes
			subDrts := buildDocRouterPaths(subRoutes, rt.Pattern)
			for k, v := range subDrts {
				newPath := strings.ReplaceAll(rt.Pattern+k, "/*/", "/")
				newPath = strings.TrimSuffix(newPath, "/")
				drts[newPath] = v
			}

		} else {
			hall := rt.Handlers["*"]

			// sort route-handler map
			keys := []string{}
			for method, _ := range rt.Handlers {
				keys = append(keys, method)
			}
			sort.Strings(keys)

			for _, method := range keys {
				h := rt.Handlers[method]
				if method != "*" && hall != nil && fmt.Sprintf("%v", hall) == fmt.Sprintf("%v", h) {
					continue
				}

				var endpoint http.Handler
				chain, _ := h.(*chi.ChainHandler)

				if chain != nil {
					endpoint = chain.Endpoint
				} else {
					endpoint = h
				}
				path := strings.ReplaceAll(prefix+rt.Pattern, "/*/", "/")
				drt[strings.ToLower(method)] = buildFuncInfo(endpoint, path, strings.ToLower(method), len(keys))
			}

			drts[rt.Pattern] = drt
		}

	}

	return drts
}

func buildFuncInfo(i interface{}, path string, method string, maxForward int) FuncInfo {

	fi := FuncInfo{}

	if strings.Contains(getCallerFrame(i).Func.Name(), helperFuncName) {
		if helperFuncs[0].Method != method {
			for c := 0; c < maxForward; c++ {
				if helperFuncs[c].Method == method {
					i = helperFuncs[c].Func
					helperFuncs = append(helperFuncs[:c], helperFuncs[c+1:]...)
					break
				}
			}
		} else {
			i = helperFuncs[0].Func
			helperFuncs = helperFuncs[1:]
		}
	}

	var parameters []Parameter
	re := regexp.MustCompile("{.*}")
	foundParams := re.FindAllString(path, -1)
	for _, p := range foundParams {
		p := strings.TrimSuffix(strings.TrimPrefix(p, "{"), "}")
		t := "string"
		if strings.HasSuffix(p, ":[0-9]+") {
			t = "number"
		}
		parameters = append(parameters, Parameter{
			Name:     p,
			In:       "path",
			Required: true,
			Schema: ParamSchema{
				Type: t,
			},
		})
	}
	fi.Parameters = parameters

	if authActivated {
		fi.Security = []map[string][]string{{
			"bearerAuth": []string{},
		}}
	} else {
		fi.Security = []map[string][]string{{}}
	}

	frame := getCallerFrame(i)

	funcPath := frame.Func.Name()

	pkgName := getPkgName(frame.File)
	idx := strings.Index(funcPath, "/"+pkgName)
	if idx > 0 {
		fi.Summary = funcPath[idx+2+len(pkgName):]
	} else {
		fi.Summary = funcPath
	}
	fi.Summary = strings.Split(fi.Summary, ".")[len(strings.Split(fi.Summary, "."))-1]
	fi.Summary = strings.TrimSuffix(fi.Summary, "-fm")
	comment := getFuncComment(frame.File, frame.Line)
	if comment == "" {
		comment = getFuncComment(frame.File, frame.Line-1)
	}
	fi.Responses = map[string]Response{}
	finalCommentLines := []string{}
	for _, commentLine := range strings.Split(comment, "\n") {
		if commentLine == "" {
			continue
		}
		if strings.HasPrefix(commentLine, "swago.response: ") {
			responseRefs := strings.ReplaceAll(strings.TrimPrefix(commentLine, "swago.response: "), " ", "")
			for _, responseRef := range strings.Split(responseRefs, ",") {
				if responseStates[responseRef] == "default" {
					continue
				}
				desc := ""
				if ok, err := strconv.Atoi(responseStates[responseRef]); err == nil {
					desc = http.StatusText(ok)
				}
				fi.Responses[responseStates[responseRef]] = Response{

					Description: desc,
					Content: Content{
						ContentType: ContentType{
							Schema: Schema{
								Ref: "#/components/schemas/" + responseRef,
							},
						},
					},
				}
			}
		} else if strings.HasPrefix(commentLine, "swago.request: ") {
			requestRef := strings.TrimPrefix(commentLine, "swago.request: ")
			required := false
			requestRef1 := strings.Split(requestRef, ",")[0]
			if strings.Contains(requestRef1, "*") {
				requestRef1 = strings.ReplaceAll(requestRef1, "*", "")
				required = true
			}
			reqDesc := ""
			if len(strings.Split(requestRef, ",")) > 1 {
				reqDesc = strings.Split(requestRef, ",")[1]
			}
			fi.RequestBody = &RequestBody{
				Description: reqDesc,
				Required:    required,
				Content: Content{
					ContentType: ContentType{
						Schema: Schema{
							Ref: "#/components/schemas/" + requestRef1,
						},
					},
				},
			}
		} else if strings.HasPrefix(commentLine, "swago.query: ") {
			params := extractParam("query", commentLine)
			if params != nil {
				fi.Parameters = append(fi.Parameters, params...)
			}
		} else if strings.HasPrefix(commentLine, "swago.header: ") {
			params := extractParam("header", commentLine)
			if params != nil {
				fi.Parameters = append(fi.Parameters, params...)
			}
		} else if strings.HasPrefix(commentLine, "swago.tag: ") {
			tags := strings.ReplaceAll(strings.TrimPrefix(commentLine, "swago.tag: "), " ", "")
			if len(tags) > 0 {
				fi.Tags = strings.Split(tags, ",")
			}
		} else {
			finalCommentLines = append(finalCommentLines, commentLine)
		}
	}
	defaultResponse := ""
	for k, v := range responseStates {
		if v == "default" {
			defaultResponse = k
		}
	}

	if _, ok := fi.Responses["default"]; !ok && defaultResponse != "" {
		fi.Responses["default"] = Response{
			Description: "Default Response",
			Content: Content{
				ContentType: ContentType{
					Schema: Schema{
						Ref: "#/components/schemas/" + defaultResponse,
					},
				},
			},
		}
	}
	fi.Description = strings.Join(finalCommentLines, "\n")
	return fi
}

func extractParam(paramType, commentLine string) []Parameter {
	re := regexp.MustCompile("{.*}")
	var parameters []Parameter

	params := strings.ReplaceAll(strings.TrimPrefix(commentLine, "swago."+paramType+": "), " ", "")
	for _, param := range strings.Split(params, ",") {
		var enumValues []string
		if res := re.FindString(param); res != "" {
			param = strings.ReplaceAll(param, res, "")
			res = strings.TrimSuffix(strings.TrimPrefix(res, "{"), "}")
			enumValues = append(enumValues, strings.Split(res, ";")...)
		}
		t := "string"
		required := false
		if strings.Contains(param, "+") {
			param = strings.ReplaceAll(param, "+", "")
			t = "number"
		}
		if strings.Contains(param, "*") {
			param = strings.ReplaceAll(param, "*", "")
			required = true
		}
		parameters = append(parameters, Parameter{
			Name:     param,
			In:       paramType,
			Required: required,
			Schema: ParamSchema{
				Type: t,
				Enum: enumValues,
			},
		})
	}
	return parameters
}
