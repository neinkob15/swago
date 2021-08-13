# Swago

Swago is a helper for the go-chi router to automatically generate swagger files.

### Comments for handler functions: 
|Comment|Description|
|:---|:---|
|``` // swago.response: StructName```|Defines the response type. Only use the types that you have declared with ```swago.RegisterType(...)``` (see below)|
|``` // swago.request: StructName*,Description```|Defines the request type (```swago.RegisterType(...)```). Star is for required. After comma you can add a description.|
|``` // swago.tag: TagName ```|Defines the swagger tag (section) to use. Needs to be defined by ```swago.RegisterTag(...)```. |
|``` // swago.query: type{left;right}*,id+* ```|Defines query parameters. Star is for required, plus is for numerical. Curly brackets define an enum (in this case, query parameter with name "type" can have values "left" and "right").
|``` // swago.header: header-name+* ```|Defines a header parameter. Star is for required, plus is for numerical.|

### Example Tags for structs:
|Example|Description|
|:---|:---|
|``` `swago:"Description of a struct field,required,readOnly"` ```|Defines the description,"required" specifies required fields, "readOnly" defines readOnly-fields.|
|``` `swago:"Description2,writeOnly"` ```|"writeOnly" defines "writeOnly"-fields of the type

### Functions:
|Function|Description|
|:---|:---|
|``` swago.RegisterType("cat", 200, reflect.ValueOf(Cat{})) ```|Registers the type of struct "Cat" as name "cat" for the swagger-file and associates it with the http-code 200. This functions reads the 'swago' struct-tags.|
|``` swago.RegisterDefaultResponse("err1", reflect.ValueOf(HttpError{})) ```|With this function you can register a default response, that is assigned to every endpoint.|
|``` swago.RegisterTag("TagName", "Tag-Description") ```|Here you can register a swagger-tag with a description. Used in combination with ``` // swago.tag: ... ```|
|``` swago.RegisterServer("http://localhost:8081", "Server-Description") ```|Specifies the server to use for the documentation file|
|``` swago.ActivateJWTAuth() ```| Adds the JWT-Bearer Authentication to the swagger-file|
|``` swago.RegisterHelper(passedFunction) ```|If you have a helper function for the handler functions (like a middleware, f.e. ```router.Get("/endpoint", helperFunction(actualHandler))``` you have to paste this code into the helper function to correctly identify all endpoints and handlers.|
|``` swago.SwaggerRoutesDoc(chiRouter, "Title", "Description") ```|This returns the final swagger.yaml documentation as a string that you can print to a file|