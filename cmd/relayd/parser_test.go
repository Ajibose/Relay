package main

import (
	"reflect"
	"testing"
)

// Request test cases
var validRequest = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent: curl\r\n\r\nsuccess")
var missingBounday = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent: curl")
var malformedRequestLine = []byte("GET /user HTTP/1 1\r\nHost: localhost\r\nAgent: curl\r\n\r\n")
var malformedHeader = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent curl\r\n\r\n")

// Response test cases
var validReponse = []byte("HTTP/1.1 201 Created\r\nContent-Type: application/json\r\n\r\n\"id\": \"usr_98765\",\"name\": \"Jane Doe\"")
var responseMissingBoundary = []byte("HTTP/1.1 201 Created\r\nContent-Type: application/json \"id\": \"usr_98765\"")
var malformedStatusLine = []byte("HTTP/1.1 201 Created Successfuly\r\nContent-Type: application/json\r\n\r\n\"id\": \"usr_98765\"")
var malformedResponseHeader = []byte("HTTP/1.1 201 Created\r\nContent-Type application/json\r\n\r\n\"id\": \"usr_98765\",\"name\": \"Jane Doe\"")

func TestParseRequest(t *testing.T) {
	cases := []struct {
		buf             []byte
		expectedParsed  *Request
		expectedInt     int
		descriptionName string
	}{
		{
			validRequest,
			&Request{method: "GET", requestTarget: "/user", headers: map[string]string{"Host": "localhost", "Agent": "curl"}, body: []byte("success")},
			0,
			"Valid Request",
		},
		{missingBounday, nil, -1, "Missing Boundary"},
		{malformedRequestLine, nil, -1, "Malformed Request Line"},
		{[]byte(""), nil, -1, "Empty Request Buffer"},
		{malformedHeader, nil, -1, "Header without colon"},
	}

	for _, c := range cases {
		t.Run(c.descriptionName, func(t *testing.T) {
			gotParsed, gotInt := parseRequest(c.buf)
			if gotInt != c.expectedInt {
				t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotInt, c.expectedInt)
			}

			if c.expectedParsed == nil && gotParsed != nil {
				t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotParsed, c.expectedParsed)
			}

			if gotParsed != nil && c.expectedParsed != nil {
				if gotParsed.requestTarget != c.expectedParsed.requestTarget ||
					gotParsed.method != c.expectedParsed.method ||
					!reflect.DeepEqual(gotParsed.headers, c.expectedParsed.headers) ||
					!reflect.DeepEqual(gotParsed.body, c.expectedParsed.body) {
					t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotParsed, c.expectedParsed)
				}
			}

		})
	}
}

func TestParseResponse(t *testing.T) {
	cases := []struct {
		buf             []byte
		expectedParsed  *Response
		expectedInt     int
		descriptionName string
	}{
		{
			validReponse,
			&Response{headers: map[string]string{"Content-Type": "application/json"}, body: []byte("\"id\": \"usr_98765\",\"name\": \"Jane Doe\""), statusCode: 201},
			0,
			"Valid Response",
		},
		{responseMissingBoundary, nil, -1, "Missing Boundary"},
		{malformedStatusLine, nil, -1, "Nalformed Status Line"},
		{malformedResponseHeader, nil, -1, "Malformed Response Header"},
		{[]byte(""), nil, -1, "Empty Response Buffer"},
	}

	for _, c := range cases {
		t.Run(c.descriptionName, func(t *testing.T) {
			gotParsed, gotInt := parseResponse(c.buf)

			if gotInt != c.expectedInt {
				t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotInt, c.expectedInt)
			}

			if c.expectedParsed == nil && gotParsed != nil {
				t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotParsed, c.expectedParsed)
			}

			if gotParsed != nil && c.expectedParsed != nil {
				if gotParsed.statusCode != c.expectedParsed.statusCode ||
					!reflect.DeepEqual(gotParsed.body, c.expectedParsed.body) ||
					!reflect.DeepEqual(gotParsed.headers, c.expectedParsed.headers) {
					t.Fatalf("%v: Got %v, expected %v", c.descriptionName, gotParsed, c.expectedParsed)
				}
			}
		})
	}
}
