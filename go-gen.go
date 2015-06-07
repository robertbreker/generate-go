package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type Param struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Doc  string `json:"doc"`
}

type Message struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Result      []string `json:"result"`
	Params      []Param  `json:"params"`
}

type EnumValue struct {
	Name string `json:"name"`
}

type Enum struct {
	Name   string      `json:"name"`
	Values []EnumValue `json:"values"`
}

type Objecttype struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Messages    []Message `json:"messages"`
	Enums       []Enum    `json:"enums"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func handleparam(param Param) (code string) {
	switch strings.ToLower(param.Type) {
	case "bool":
		fallthrough
	case "string":
		code += variableName(param.Name) + " " + strings.ToLower(param.Type)
	case "string set":
		code += variableName(param.Name) + " []string"
	case "int":
		code += variableName(param.Name) + " uint64"
	case "(string -> string) map":
		code += variableName(param.Name) + " map[string]string"
	default:
		if strings.Contains(param.Type, " ref set") {
			paramType := makeObjectName(strings.Replace(param.Type, " ref set", "", -1))
			code += variableName(param.Name) + " []" + paramType
		} else if strings.Contains(param.Type, " ref -> ") && strings.Contains(param.Type, " map") {
			code += variableName(param.Name) + " map[string]string"
		} else if strings.Contains(param.Type, " ref") {
			paramType := makeObjectName(strings.Replace(param.Type, " ref", "", -1))
			code += variableName(param.Name) + " " + paramType
		} else if strings.Contains(param.Type, "enum ") {
			paramType := makeObjectName(strings.Replace(param.Type, "enum ", "", -1))
			code += variableName(param.Name) + " " + paramType
		} else if strings.Contains(param.Type, " set") {
			paramType := makeObjectName(strings.Replace(param.Type, " set", "", -1))
			code += variableName(param.Name) + " " + paramType
		}
	}
	return
}

func CamelCase(name string) (resultString string) {
	nextCapital := true
	for _, character := range name {
		characterString := string(character)
		if characterString == "_" {
			nextCapital = true
			continue
		}

		if nextCapital {
			characterString = strings.Title(characterString)
			nextCapital = false
		}
		resultString += characterString
	}
	return
}

func variableName(name string) (resultString string) {
	resultString = makeObjectName(name)
	// Convert the first character to lower case
	resultBytes := []rune(resultString)
	resultBytes[0] = unicode.ToLower(resultBytes[0])
	resultString = string(resultBytes)
	return resultString
}

func packageName(name string) (resultString string) {
	resultString = strings.ToLower(strings.Replace(name, "_", "", -1))
	return
}

func addApiCall(objecttype Objecttype, message Message, errorstring string) (code string) {
	for _, param := range message.Params {
		if param.Type == "(string -> string) map" || strings.Contains(param.Type, " ref -> string) map") {
			code += "	" + variableName(param.Name) + "_xmlstruct := make(xmlrpc.Struct)\n"
			code += "	for ekey, evalue := range " + variableName(param.Name) + " {\n"
			code += "		" + variableName(param.Name) + "_xmlstruct[ekey] = evalue\n"
			code += "	}\n"
		}
	}
	code += "	aResult := APIResult{}\n"
	code += "	err = self.Client.APICall(&aResult, \"" + objecttype.Name + "." + message.Name + "\", self.Ref"
	for _, param := range message.Params {
		if param.Name != "session_id" && param.Name != "self" && param.Name != objecttype.Name {
			if param.Type == "(string -> string) map" {
				code += ", " + variableName(param.Name) + "_xmlstruct"
			} else if param.Type == objecttype.Name+" ref" {
				//code += ", self.Ref"
			} else if strings.Contains(param.Type, " ref") {
				code += ", " + variableName(param.Name) + ".Ref"
			} else if param.Type == "int" {
				code += ", fmt.Sprintf(\"%d\", " + variableName(param.Name) + ")"
			} else {
				code += ", " + variableName(param.Name)
			}
		}
	}
	code += ")\n"
	code += "	if err != nil {\n"
	code += "		return "
	if errorstring != "_" {
		code += errorstring + ", "
	}
	code += "err\n"
	code += "	}\n"
	return code
}

func handlemessage(objecttype Objecttype, message Message) (code string) {
	if objecttype.Name == "task" {
		if message.Name == "get_result" {
			code = ""
			return code
		}
	}
	code += "func "
	code += "(self *" + makeObjectName(objecttype.Name) + ") "
	code += CamelCase(message.Name) + "("
	paramcode := []string{}
	for _, param := range message.Params {
		if param.Name == "session_id" || param.Name == "self" || param.Name == objecttype.Name || param.Type == objecttype.Name+" ref" {
			continue
		}
		paramcode = append(paramcode, handleparam(param))
	}
	code += strings.Join(paramcode, ", ")
	code += ") "
	if message.Result[0] == "string" || message.Result[0] == "float" || message.Result[0] == "int" {
		returntype := ""
		if message.Result[0] == "string" {
			returntype = "string"
		} else if message.Result[0] == "float" {
			returntype = "float64"
		} else if message.Result[0] == "int" {
			returntype = "int"
		}

		varname := "response"
		if strings.Contains(message.Name, "get_") {
			varname = variableName(strings.Replace(message.Name, "get_", "", -1))
		}
		code += "(" + varname + " " + returntype + ", err error) {\n"

		for _, param := range message.Params {
			if param.Type == "(string -> string) map" {
				code += "	" + param.Name + "_rec := make(xmlrpc.Struct)\n"
				code += "	for key, value := range params {\n"
				code += "		" + param.Name + "_rec[key] = value\n"
				code += "	}\n"
			}
		}

		code += addApiCall(objecttype, message, varname)

		code += "	" + varname + " = aResult.Value.(" + returntype + ")\n"
		code += "	return " + varname + ", nil\n"
	} else if message.Result[0] == objecttype.Name+" record" {
		code += "(recordMap map[string]string, err error) {\n"
		code += "	recordMap = make(map[string]string, 0)\n"

		code += addApiCall(objecttype, message, "recordMap")

		code += "	for k, v := range aResult.Value.(xmlrpc.Struct) {\n"
		code += "		recordMap[k] = v.(string)\n"
		code += "	}\n"
		code += "	return recordMap, nil\n"
	} else if strings.Contains(message.Result[0], " ref -> string set) map") {
		reference := strings.Split(message.Result[0], " ref -> string set")[0]
		variablenamesingle := variableName(reference)
		variablenamearray := variablenamesingle + "s"

		code += "(" + variablenamearray + " []map[string][]string, err error) {\n"
		code += "       " + variablenamearray + " = make([]map[string][]string, 0)\n"
		code += addApiCall(objecttype, message, variablenamearray)
		code += "       for _, element := range aResult.Value.(xmlrpc.Struct) {\n"
		code += "               for key, value := range element.Value.([]interface{}) {\n"
		code += "                       " + variablenamearray + "[key] = append(" + variablenamearray + "[key], value.(string))\n"
		code += "               }\n"
		code += "       }\n"
		code += "       return " + variablenamearray + ", nil\n"

	} else if strings.Contains(message.Result[0], " -> string) map") {
		varname := variableName(strings.Replace(message.Name, "get_", "", -1))

		code += "(" + varname + "Map map[string]string, err error) {\n"
		code += "	" + varname + "Map = make(map[string]string, 0)\n"

		code += addApiCall(objecttype, message, varname+"Map")

		code += "	for k, v := range aResult.Value.(xmlrpc.Struct) {\n"
		code += "		" + varname + "Map[k] = v.(string)\n"
		code += "	}\n"
		code += "	return " + varname + "Map, nil\n"
	} else if strings.Contains(message.Result[0], "(string -> ") && strings.Contains(message.Result[0], " ref) map") {
		varname := variableName(strings.Replace(message.Name, "get_", "", -1))
		reference := makeObjectName(strings.Replace(strings.Replace(message.Result[0], "(string -> ", "", -1), " ref) map", "", -1))

		code += "(" + varname + "Map map[string]" + reference + ", err error) {\n"
		code += "       " + varname + "Map = make(map[string]" + reference + ", 0)\n"

		code += addApiCall(objecttype, message, varname)

		code += "       for k, v := range aResult.Value.(xmlrpc.Struct) {\n"
		code += "       	" + varname + "Map[k] = make([]" + reference + ", 0)\n"
		code += "               " + varname + "Map[k] = " + reference + "{}\n"
		code += "               " + varname + "Map[k].Ref = v.(string)\n"
		code += "               " + varname + "Map[k].Client = self.Client\n"
		code += "       }\n"
		code += "       return " + varname + "Map, nil\n"
	} else if message.Result[0] == "void" {
		code += "(err error) {\n"

		code += addApiCall(objecttype, message, "_")

		code += "	return err\n"
	} else if strings.Contains(message.Result[0], " ref set") {
		reference := makeObjectName(strings.Replace(message.Result[0], " ref set", "", -1))
		variablenamesingle := variableName(reference)
		variablenamearray := variablenamesingle + "s"
		code += "(" + variablenamearray + " []" + reference + ", err error) {\n"
		code += "       " + variablenamearray + " = make([]" + reference + ", 0)\n"
		code += addApiCall(objecttype, message, variablenamearray)
		code += "	for _, elem := range aResult.Value.([]interface{}) {\n"
		code += "		" + variablenamesingle + " := " + reference + "{}\n"
		code += "		" + variablenamesingle + ".Ref = elem.(string)\n"
		code += "		" + variablenamesingle + ".Client = self.Client\n"
		code += "		" + variablenamearray + " = append(" + variablenamearray + ", " + variablenamesingle + ")\n"
		code += "	}\n"
		code += "	return " + variablenamearray + ", nil\n"
	} else if strings.Contains(message.Result[0], " ref -> ") && strings.Contains(message.Result[0], " record) map") {
		reference := strings.Split(message.Result[0], " ref ->")[0]
		variablenamesingle := variableName(reference)
		variablenamearray := variablenamesingle + "s"

		code += "(" + variablenamearray + " []map[string]string, err error) {\n"
		code += "       " + variablenamearray + " = make([]map[string]string, 0)\n"
		code += addApiCall(objecttype, message, variablenamearray)
		code += "	for _, element := range aResult.Value.(xmlrpc.Struct) {\n"
		code += "		" + variablenamesingle + " := make(map[string]string)\n"
		code += "		for k, v := range element.(xmlrpc.Struct) {\n"
		code += "			" + variablenamesingle + "[k] = v.(string)\n"
		code += "		}\n"
		code += "		" + variablenamearray + " = append(" + variablenamearray + ", " + variablenamesingle + ")\n"
		code += "	}\n"
		code += "	return " + variablenamearray + ", nil\n"
	} else if strings.Contains(message.Result[0], " ref") {
		reference := variableName(strings.Replace(message.Result[0], " ref", "", -1))
		code += "(" + reference + " *" + makeObjectName(reference) + ", err error) {\n"

		code += addApiCall(objecttype, message, "nil")

		code += "	" + reference + " = new(" + strings.Title(reference) + ")\n"
		code += "	" + reference + ".Ref = aResult.Value.(string)\n"
		code += "	" + reference + ".Client = self.Client\n"
		code += "	return " + reference + ", nil\n"
	} else if message.Result[0] == "string set" {
		varname := variableName(strings.Replace(message.Name, "get_", "", -1))
		code += "(" + varname + " []string, err error) {\n"
		code += "       " + varname + " = make([]string, 0)\n"
		code += addApiCall(objecttype, message, varname)
		code += "	" + varname + " = make([]string, 0)\n"
		code += "	for _, elem := range aResult.Value.([]interface{}) {\n"
		code += "		" + varname + " = append(" + varname + ", elem.(string))\n"
		code += "	}\n"
		code += "	return " + varname + ", nil\n"
	} else if strings.Contains(message.Result[0], "(string -> enum ") {
		rawenumtype := strings.Split(message.Result[0], "enum ")[1]
		code += "(status map[string]" + rawenumtype + ", err error) {\n"
		code += addApiCall(objecttype, message, "status")
		code += "        rawStatus := strings.ToLower(aResult.Value.(string))\n"
		code += "        switch rawStatus {\n"
		for _, enum := range objecttype.Enums {
			if enum.Name == rawenumtype {
				for _, value := range enum.Values {
					code += "        case \"" + value.Name + "\":\n"
					code += "                status = " + makeObjectName(enum.Name) + makeObjectName(value.Name) + "\n"
				}
			}
		}
		code += "        default:\n"
		code += "                panic(fmt.Sprintf(\"Task.get_status: Unknown status '%s'\", rawStatus))\n"
		code += "        }\n"
		code += "        return status, nil\n"
	} else if strings.Contains(message.Result[0], "enum ") {
		rawenumtype := strings.Split(message.Result[0], "enum ")[1]
		code += "(status " + rawenumtype + ", err error) {\n"
		code += addApiCall(objecttype, message, "status")
		code += "        rawStatus := strings.ToLower(aResult.Value.(string))\n"
		code += "        switch rawStatus {\n"
		for _, enum := range objecttype.Enums {
			if enum.Name == rawenumtype {
				for _, value := range enum.Values {
					code += "        case \"" + value.Name + "\":\n"
					code += "                status = " + makeObjectName(enum.Name) + makeObjectName(value.Name) + "\n"
				}
			}
		}
		code += "        default:\n"
		code += "                panic(fmt.Sprintf(\"Task.get_status: Unknown status '%s'\", rawStatus))\n"
		code += "        }\n"
		code += "        return status, nil\n"
	} else {
		code = ""
		return
	}
	code += "}\n\n"
	return
}

func makeObjectName(objectName string) (returnObjectName string) {
	returnObjectName = CamelCase(objectName)
	// Strip none-alpha-numerical
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		panic(err)
	}
	returnObjectName = reg.ReplaceAllString(returnObjectName, "")
	switch returnObjectName {
	case "Type":
		returnObjectName = "aType"
	case "Interface":
		returnObjectName = "aInterface"
	}
	return returnObjectName
}

func handleEnum(objecttype Objecttype, enum Enum) (code string) {
	enumtypename := makeObjectName(enum.Name)
	code += "type " + enumtypename + " int\n\n"
	code += "const (\n"
	code += "	_ " + enumtypename + " = iota\n"
	for _, value := range enum.Values {
		code += "	" + enumtypename + makeObjectName(value.Name) + "\n"
	}
	code += ")\n\n"
	return
}

func handleobjecttype(objecttype Objecttype) {
	if len(objecttype.Messages) > 1 {
		code := ""
		code += "package client\n"
		code += "\n"
		code += "import (\n"
		code += "	\"github.com/nilshell/xmlrpc\"\n"
		code += ")\n"
		code += "\n"
		code += "type " + makeObjectName(objecttype.Name) + " XenAPIObject\n\n"

		for _, enum := range objecttype.Enums {
			code += handleEnum(objecttype, enum)
		}

		for _, message := range objecttype.Messages {
			code += handlemessage(objecttype, message)
		}
		extracode, err := ioutil.ReadFile("extras/" + packageName(objecttype.Name) + ".go")
		d1 := []byte(code + string(extracode))
		err = ioutil.WriteFile("generated/"+packageName(objecttype.Name)+".go", d1, 0644)
		check(err)
	}
}

func main() {
	content, err := ioutil.ReadFile("xenapi.json")
	check(err)
	var conf []Objecttype
	err = json.Unmarshal(content, &conf)
	check(err)
	os.RemoveAll("generated")
	os.Mkdir("generated/", 0755)
	for _, element := range conf {
		handleobjecttype(element)
	}
	os.Link("extras/client.go", "generated/client.go")
}
