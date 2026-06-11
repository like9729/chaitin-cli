package parser

// Spec is a subset of OpenAPI 3.0 fields needed for CLI generation.
type Spec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type PathItem struct {
	Post *Operation `json:"post,omitempty"`
	Get  *Operation `json:"get,omitempty"`
	Put  *Operation `json:"put,omitempty"`
}

type Operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Tags        []string            `json:"tags"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type RequestBody struct {
	Content  map[string]MediaType `json:"content"`
	Required bool                 `json:"required"`
}

type MediaType struct {
	Schema *Schema `json:"schema"`
}

type Schema struct {
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Properties   map[string]Schema `json:"properties"`
	Required     []string          `json:"required"`
	Items        *Schema           `json:"items,omitempty"`
	Enum         []interface{}     `json:"enum,omitempty"`
	Default      interface{}       `json:"default,omitempty"`
	Ref          string            `json:"$ref,omitempty"`
	AllOf        []Schema          `json:"allOf,omitempty"`
	Minimum      *float64          `json:"minimum,omitempty"`
	ExclusiveMin *bool             `json:"exclusiveMinimum,omitempty"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

// CommandSpec represents a flattened API operation ready for command generation.
type CommandSpec struct {
	OperationID string
	Summary     string
	Tags        []string
	Path        string
	Method      string
	BodyParams  []ParamSpec
	HasBody     bool
}

// ParamSpec represents a single body parameter for flag generation.
type ParamSpec struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
	Enum        []interface{}
	Minimum     *float64
}
