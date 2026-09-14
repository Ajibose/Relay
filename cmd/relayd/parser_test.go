package main

import (
	"reflect"
	"testing"
)

var validRequest = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent: curl\r\n\r\nsuccess")
var missingBounday = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent: curl")
var malformedRequestLine = []byte("GET /user HTTP/1 1\r\nHost: localhost\r\nAgent: curl\r\n\r\n")
var malformedHeader = []byte("GET /user HTTP/1.1\r\nHost: localhost\r\nAgent curl\r\n\r\n")

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
