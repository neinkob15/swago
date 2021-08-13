# Swago

Swago is a helper for the go-chi router to automatically generate swagger files.

Example Comments for handler functions: 
- ``` // swago.response: StructName```
- ``` // swago.request: StructName*,Description``` 
- ``` // swago.tag: TagName ```
- ``` // swago.query: type{left;right}*,id+* ```
- ``` // swago.header: header-name+* ```

Example Tags for structs:
- ``` `swago:"Description of a struct field,required,readOnly"` ```
- ``` `swago:"Description2,writeOnly"` ```

Example Functions for code:
- ``` swago.RegisterType("cat", 200, reflect.ValueOf(Cat{})) ```
- ``` swago.RegisterDefaultResponse("err1", reflect.ValueOf(HttpError{})) ```
- ``` swago.RegisterTag("TagName", "Tag-Description") ```
- ``` swago.RegisterServer("http://localhost:8081", "Server-Description") ```
- ``` swago.ActivateJWTAuth() ```
- ``` swago.RegisterHelper(passedFunction) ``` 
- ``` swago.JSONRoutesDoc(chiRouter, "Title", "Description") ```