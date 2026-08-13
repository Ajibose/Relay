package main

import (
	"bytes"
	"strconv"
	"strings"
)

// Request holds the parsed pieces of an HTTP request line and headers.
// body is not bounded by Content-Length - see parked_decisions.md for the
// keep-alive/pipelining gap this leaves.
type Request struct {
	method        string
	requestTarget string
	headers       map[string]string
	body          []byte
}

// Response holds the parsed pieces of an HTTP status line and headers.
// Same body caveat as Request.
type Response struct {
	headers    map[string]string
	body       []byte
	statusCode int
}

// parseRequest parses buf as a single HTTP request: request line, headers,
// then everything after the blank line as body. Returns (nil, -1) if the
// request line or headers are malformed, or no \r\n\r\n boundary is found
// at all.
//
// body is just "everything left in buf" rather than sliced to
// Content-Length, so this assumes exactly one request per buf. See
// parked_decisions.md for what breaks if a connection sends more than one.
func parseRequest(buf []byte) (*Request, int) {
	var requestStruct Request

	requestLineIndex := bytes.Index(buf, []byte("\r\n"))
	if requestLineIndex == -1 {
		return nil, -1
	}

	requestLine := buf[:requestLineIndex]
	requestLineSlice := bytes.Split(requestLine, []byte(" "))
	if len(requestLineSlice) != 3 {
		return nil, -1
	}

	requestStruct.method = string(requestLineSlice[0])
	requestStruct.requestTarget = string(requestLineSlice[1])

	headerIndexStart := requestLineIndex + 2
	headerIndexEnd := bytes.Index(buf, []byte("\r\n\r\n"))
	if headerIndexEnd == -1 {
		return nil, -1
	}

	var err int
	requestStruct.headers, err = parseHeader(buf[headerIndexStart:headerIndexEnd])
	if err != 0 {
		return nil, err
	}

	requestStruct.body = buf[headerIndexEnd+4:]

	return &requestStruct, 0
}

// parseResponse parses buf as a single HTTP response: status line, headers,
// then everything after the blank line as body. S
//
// body is just "everything left in buf" rather than sliced to
// Content-Length, so this assumes exactly one response per buf.
func parseResponse(buf []byte) (*Response, int) {
	var responseStruct Response

	statusLineIndex := bytes.Index(buf, []byte("\r\n"))
	if statusLineIndex == -1 {
		return nil, -1
	}

	statusLine := buf[:statusLineIndex]
	statusLineSlice := bytes.Split(statusLine, []byte(" "))
	if len(statusLineSlice) != 3 {
		return nil, -1

	}

	status, parseErr := strconv.Atoi(string(statusLineSlice[1]))
	if parseErr != nil {
		return nil, -1
	}

	responseStruct.statusCode = status

	headerIndexStart := statusLineIndex + 2
	headerIndexEnd := bytes.Index(buf, []byte("\r\n\r\n"))
	if headerIndexEnd == -1 {
		return nil, -1
	}

	var err int
	responseStruct.headers, err = parseHeader(buf[headerIndexStart:headerIndexEnd])
	if err != 0 {
		return nil, err
	}

	responseStruct.body = buf[headerIndexEnd+4:]

	return &responseStruct, 0
}

// parseHeader splits a header block - everything between the start line and
// the blank line, not including either - into a map of header name to
// value. Shared by parseRequest and parseResponse since headers look
// identical either way; this function never sees the start line that came
// before it.
func parseHeader(buf []byte) (map[string]string, int) {
	headerSlice := bytes.Split(buf, []byte("\r\n"))
	headerMap := make(map[string]string)

	for _, header := range headerSlice {
		fieldSlice := bytes.SplitN(header, []byte(":"), 2)
		if len(fieldSlice) != 2 {
			return nil, -1
		}

		fieldKey := strings.TrimSpace(string(fieldSlice[0]))
		fieldValue := strings.TrimSpace(string(fieldSlice[1]))

		headerMap[fieldKey] = fieldValue

	}

	return headerMap, 0
}
