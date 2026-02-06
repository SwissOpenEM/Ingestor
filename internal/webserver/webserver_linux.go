//go:build linux

package webserver

// Copy the openapi specs to local folder so it can be embedded in order to statically serve it
//go:generate cp ../../api/openapi.yaml ./openapi.yaml
