package rpccodecs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gorilla/rpc"
)

type CustomRequestsCodec struct {
}

func NewCustomRequestsCodec() *CustomRequestsCodec {
	return &CustomRequestsCodec{}
}

func (c *CustomRequestsCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	outerCR := &CustomRequestsCodecRequest{}
	jsonC := NewCodec()
	innerCR := jsonC.NewRequest(r)
	outerCR.CodecRequest = innerCR.(*CodecRequest)
	return outerCR
}

type CustomRequestsCodecRequest struct {
	*CodecRequest
}

func (c *CustomRequestsCodecRequest) Method() (string, error) {
	m, err := c.CodecRequest.Method()
	if len(m) > 1 && err == nil {
		parts := strings.Split(m, "_")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid method: %s", m)
		}
		service, method := parts[0], parts[1]
		return capitalize(service) + "." + capitalize(method), nil
	}
	return m, err
}

func capitalize(value string) string {
	r, n := utf8.DecodeRuneInString(value)
	return string(unicode.ToUpper(r)) + value[n:]
}

var null = json.RawMessage([]byte("null"))

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

// serverRequest represents a JSON-RPC request received by the server.
type serverRequest struct {
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// An Array of objects to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	Id *json.RawMessage `json:"id"`
}

// serverResponse represents a JSON-RPC response returned by the server.
type serverResponse struct {
	// The Object that was returned by the invoked method. This must be null
	// in case there was an error invoking the method.
	Result interface{} `json:"result"`
	// An Error object if there was an error invoking the method. It must be
	// null if there was no error.
	Error interface{} `json:"error"`
	// This must be the same id as the request it is responding to.
	Id *json.RawMessage `json:"id"`
}

// ----------------------------------------------------------------------------
// Codec
// ----------------------------------------------------------------------------

// NewCodec returns a new JSON Codec.
func NewCodec() *Codec {
	return &Codec{}
}

// Codec creates a CodecRequest to process each request.
type Codec struct {
}

// NewRequest returns a CodecRequest.
func (c *Codec) NewRequest(r *http.Request) rpc.CodecRequest {
	return newCodecRequest(r)
}

// ----------------------------------------------------------------------------
// CodecRequest
// ----------------------------------------------------------------------------

// newCodecRequest returns a new CodecRequest.
func newCodecRequest(r *http.Request) rpc.CodecRequest {
	// Decode the request body and check if RPC method is valid.
	req := new(serverRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	r.Body.Close()
	return &CodecRequest{request: req, err: err}
}

// CodecRequest decodes and encodes a single request.
type CodecRequest struct {
	request *serverRequest
	err     error
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *CodecRequest) Method() (string, error) {
	if c.err == nil {
		return c.request.Method, nil
	}
	return "", c.err
}

// ReadRequest fills the request object for the RPC method.
func (c *CodecRequest) ReadRequest(args interface{}) error {
	if c.err == nil {
		if c.request.Params != nil {

			if reflect.ValueOf(args).Elem().Kind() == reflect.Struct {
				params := []interface{}{args}
				c.err = json.Unmarshal(*c.request.Params, &params)
				return c.err
			}

			c.err = json.Unmarshal(*c.request.Params, &args)

		} else {
			c.err = errors.New("rpc: method request ill-formed: missing params field")
		}
	}
	return c.err
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
//
// The err parameter is the error resulted from calling the RPC method,
// or nil if there was no error.
func (c *CodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}, methodErr error) error {
	if c.err != nil {
		return c.err
	}
	res := &serverResponse{
		Result: reply,
		Error:  &null,
		Id:     c.request.Id,
	}
	if methodErr != nil {
		// Propagate error message as string.
		res.Error = methodErr.Error()
		// Result must be null if there was an error invoking the method.
		// http://json-rpc.org/wiki/specification#a1.2Response
		res.Result = &null
	}
	if c.request.Id == nil {
		// Id is null for notifications and they don't have a response.
		res.Id = &null
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		c.err = encoder.Encode(res)
	}
	return c.err
}
