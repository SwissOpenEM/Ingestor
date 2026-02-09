//go:build windows

package webserver

// Copy the openapi specs to local folder so it can be embedded in order to statically serve it
//go:generate cmd.exe /c copy ..\..\api\openapi.yaml .\openapi.yaml
