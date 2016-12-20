// Code generated by go-bindata.
// sources:
// layout.html
// assets/styles.css
// templates/factoid-info.html
// templates/factoid-list.html
// templates/home.html
// templates/invite-list.html
// templates/logs-index.html
// templates/user-manage.html
// DO NOT EDIT!

package weblogin

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data, name string) ([]byte, error) {
	gz, err := gzip.NewReader(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _layoutHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xac\x56\x5d\x77\xda\x38\x13\xbe\xcf\xaf\x50\xdd\xde\xe5\xb5\x55\x20\x49\xf3\x66\x8d\xf7\xd0\x40\x93\x34\x34\x1f\x85\xd2\xdd\xdc\xec\x91\xed\xb1\x2d\x90\x25\x22\x89\xaf\xf5\xf1\x7f\xdf\x63\xd9\x06\x42\x69\xcb\xe9\x29\x37\x96\x67\xc6\xcf\xcc\xa3\x79\x34\x22\xcb\x50\x08\x11\xe5\x80\x2c\x46\x56\x62\xa6\x2d\x94\xe7\x47\xee\xab\xee\xfd\xe5\xf0\xef\x87\x1e\x4a\x74\xca\xbc\x23\xb7\x7e\x00\x09\xbd\x23\x84\x10\xca\x32\xe4\x33\x11\x4c\x90\xa5\xa9\x66\x60\x21\x07\xe5\xb9\x6b\xd6\x5e\x96\xd1\x08\x39\xc3\x62\x9d\xe7\x59\xb6\x59\x01\x53\x90\xe7\x9f\x88\x9c\x53\x8e\xbe\x82\x8f\x6e\xb8\x06\x19\x91\x00\xb2\x0c\x78\x98\xe7\x2e\xae\x11\x10\xf0\xb0\xa8\xe4\x65\xae\x10\x22\x32\x63\xda\x56\x7a\xc5\x40\x15\x49\xed\x2a\xc6\x65\x94\x4f\x90\x04\xd6\xb6\x4a\x67\x02\xa0\x2d\x94\x48\x88\xda\x56\xa2\xf5\x54\x5d\x60\x1c\x84\x7c\xac\x9c\x80\x89\x59\x18\x31\x22\xc1\x09\x44\x8a\xc9\x98\x2c\x31\xa3\xbe\xc2\x7a\x41\xb5\x06\x69\xfb\x42\x68\xa5\x25\x99\xe2\x96\xd3\x72\xce\x70\xa0\x14\x5e\xdb\x9c\x94\x72\x27\x50\xca\x32\x69\xcb\x1f\xe5\x1a\x62\x49\xf5\xaa\x6d\xa9\x84\xb4\xce\x4f\xec\xc6\xf3\x79\x3a\xfc\x78\xdf\x19\x2c\xcf\xc7\x8d\xce\xec\x98\x9c\x7e\xed\x8e\xf8\x03\x6d\xb2\xc9\x87\x68\xb1\xe8\x75\xc8\x79\xd2\xed\x86\xe3\x27\x36\xed\x43\xbc\x4c\xc6\xa3\x4f\xbd\x46\x14\x8f\xbf\x3e\x5c\xa5\x93\x7f\xd5\x3b\x0b\x05\x52\x28\x25\x24\x8d\x29\x6f\x5b\x84\x0b\xbe\x4a\xc5\x4c\x59\xde\x6f\x63\x1b\x09\xae\x6d\xb2\x00\x25\x52\xc0\x27\xce\x3b\xe7\xad\x21\xba\x6d\x3e\x8c\xeb\x62\x1e\xfd\x35\x7d\x9e\x3e\x3d\x8d\x1e\xaf\x6e\xcf\x86\x9d\xe4\xf4\x61\xc4\xae\xee\xa3\xc7\xbb\xeb\x81\xe8\x36\x97\x7e\xef\xf8\x71\xf2\xb0\xbc\xec\x7c\x60\x77\x3d\x98\x8b\xde\x75\x6b\xc0\xde\x2a\xea\x8f\x82\xfb\xc7\x11\xbf\xfb\x19\x57\x15\x48\x3a\xd5\x48\xc9\xe0\x60\x6e\xe3\xe7\x19\xc8\x15\x6e\x3a\x4d\xa7\x55\xbd\x18\x2e\xe3\x9f\x50\xb9\x39\xfb\x70\x7a\x7f\xdb\xbb\xec\x8f\xf4\x2d\x7e\xdf\x3f\x3e\xa7\x83\x7e\xb7\x77\x2d\x16\x83\x4e\xf4\x45\xbc\x3b\x7b\xea\xff\xff\x78\x72\xd5\x89\x87\x9f\x43\xfa\x7e\x75\x73\x77\xfb\x91\x3c\x0f\x1f\xae\xf1\xe3\xe8\x6e\xd0\x18\x75\xfd\xef\x53\x71\x71\x49\x63\xbb\x7f\x2f\xdb\x55\x6c\xbc\x72\x62\x21\x62\x06\x64\x4a\x95\xa1\x14\x28\xf5\x67\x44\x52\xca\x56\xed\x0e\x83\x58\xc2\x8a\x1c\x0f\x08\x57\xc7\x83\x4b\xeb\xdb\xfe\xff\xba\x66\x30\x51\x0a\xb4\xc2\xa5\xc7\xf4\xbc\x3e\xe7\xf6\xfe\xa3\xb8\x39\x82\xc5\xc9\xae\x63\x5c\x5c\x8e\x08\xd7\x17\xe1\xca\x3b\xda\x84\x73\x32\xf7\x89\xac\x4f\xac\xcb\xc9\x1c\x05\x8c\x28\xd5\xae\x3c\xa8\x7c\xd8\x94\xcf\x41\x2a\xa8\x6b\x0e\xe9\x3a\x2e\x10\x5c\x13\xca\x41\xda\x11\x9b\xd1\x70\x4f\x44\x05\x51\x54\x00\xb2\xf2\x9b\x18\xb2\x13\xe1\x4b\xc2\xc3\x35\x75\xcb\x2b\x27\x92\x8b\x49\x85\x89\x43\x3a\xdf\x82\xa7\x61\x7b\x5d\xff\xba\x18\xc6\xc8\x54\x41\x5d\x76\xfd\xbe\x9d\x75\xc6\xb6\xd2\xd6\x81\x9c\xcc\xb7\x62\xb2\x0c\x2d\xa8\x4e\xea\x4d\x7c\x13\xcc\xa4\x04\xae\xd1\x45\x1b\x39\x77\xe6\x83\xcb\xca\x62\xfc\x92\xf0\x18\x6a\xcf\x8d\x86\x54\xd5\x7d\xd9\x02\xa4\x11\x82\xe7\x22\x28\x85\x0d\xa0\xbd\x13\xe7\x32\x5a\x17\x47\x02\x4d\xe7\x60\x79\x2e\xa9\x36\x24\xcb\x9c\x2f\x9f\xfb\x79\x6e\x79\x59\x66\x60\x8a\xb9\x4c\x3c\x17\x33\xea\xed\x24\xb3\x51\x31\xd2\xf7\xa1\xff\x2a\xdc\x46\x6a\x15\x9d\xd2\xf2\xf2\x42\x28\x9b\x34\x63\x07\x6c\x76\xbd\x94\x34\x4e\xb4\xb5\x9b\xb0\xd8\x2c\x67\xc0\x48\x30\xf9\xa2\x40\xee\xe5\xf1\xc2\x60\x8c\x34\x8d\x51\x02\x05\x5e\xdb\x6a\x35\x2d\xb4\xa0\xa1\x4e\xca\xa5\x99\x51\x59\xb6\x81\x74\x3a\x73\xa2\x89\x44\xad\x66\x9e\x5b\xdf\x40\x99\x9f\x92\x81\x02\xfd\xfd\xcf\x50\x63\xf9\x3f\xb4\xcf\x79\x76\x92\xe7\xa8\xf9\x1d\x67\xa3\x79\x9e\xe7\xe8\x64\x69\x21\xbc\x87\x42\xdd\x9b\xd7\x96\xe7\xae\x85\x10\x11\x14\x11\x5b\x15\x48\xc5\xbc\xa2\x5e\x01\xdc\xad\x04\x68\xb0\x4d\x96\x4d\x0b\x5f\xee\xd5\xaf\xcb\x63\xa0\x89\xd4\x25\x83\x52\x28\x3f\xaa\xc9\xc4\xa1\xbe\x88\xd1\x0d\xff\xa1\x8e\xf6\x28\xde\xcc\x2c\x73\x78\x85\xbc\x78\x1d\x86\xe1\x1f\x53\x12\x86\x94\xc7\xb6\x16\xd3\x8b\xc6\xe9\x74\xb9\x36\xf8\x42\x6b\x91\x96\xb6\x94\xc8\x98\xf2\x52\x42\xa5\xe5\xdb\x12\x89\x94\x62\xa1\xec\xa4\xac\x72\x5f\x51\x46\x6b\x2f\xb6\xf3\x86\x6b\x49\xfa\x22\xa6\xfc\x40\xe5\xbd\xb2\x6d\x34\x4c\xa8\x42\x69\x51\x0a\xf2\x01\x2d\xa4\xe0\x31\xb2\xed\x03\x64\xba\x7b\x7f\x3a\xb4\x48\xef\x9c\x34\x9d\x48\xe2\x99\x02\xa9\x70\x0a\x21\x9d\xa5\xff\xec\xb6\x7d\x53\x67\x9e\x3b\xe3\x29\xc4\x3f\x15\xd5\x8f\x10\x7e\xbb\x72\x0c\x78\xa5\x1c\xb3\x3e\x44\x1d\xfb\x47\xc9\xf6\xe4\x2f\x97\x2e\xe6\x64\x5e\x5c\x64\xeb\xaf\xea\x99\xfd\x26\x66\xc2\x27\xcc\xcc\xea\x72\x42\x55\x57\x5d\x71\x51\x01\xd7\x16\x72\xde\x8b\x70\xd5\x25\x9a\x98\x9b\x31\x69\x78\x9f\xa8\x52\x94\xc7\xa8\x8a\x78\xe5\xe2\xa4\x61\x2e\xc9\xdd\x21\xe7\xe2\xf2\xfe\x74\x71\xf9\xc7\x7b\xed\xf9\x2f\x00\x00\xff\xff\x9e\x23\x6e\xd9\xaf\x0b\x00\x00"

func layoutHtmlBytes() ([]byte, error) {
	return bindataRead(
		_layoutHtml,
		"layout.html",
	)
}

func layoutHtml() (*asset, error) {
	bytes, err := layoutHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "layout.html", size: 2991, mode: os.FileMode(436), modTime: time.Unix(1482220607, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsStylesCss = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x8c\x53\xd1\x6e\xd3\x30\x14\x7d\xef\x57\x5c\x6d\x0f\x93\xba\xa5\x4d\xbb\x6e\x94\x4c\x4c\x1a\x05\x09\x24\xc4\x03\x9b\x78\x41\x3c\xdc\xd8\x37\x89\x35\xc7\x37\xb3\x9d\x75\x05\xed\xdf\x91\x9d\xb4\x2b\xa5\x9d\x78\x8c\x7d\xcf\xb9\xc7\xe7\x9c\x8c\x87\xf0\x89\x50\x92\x85\xe1\x78\x90\xb3\x5c\xc1\xef\x01\x00\x40\x8e\xe2\xbe\xb4\xdc\x1a\x99\x08\xd6\x6c\x33\x38\x2e\xa8\x10\x85\xb8\x1a\x3c\x0f\x46\x06\x1f\x73\xb4\x89\xc1\xc7\x6b\xad\xae\x55\x5d\xfe\xa8\x48\x95\x95\x7f\x77\x74\x3e\x3d\xfa\xd9\x53\xd4\x68\x4b\x65\x12\x1b\x2e\x32\xb8\x68\x9e\xae\xe2\xb1\x54\xae\xd1\xb8\xca\x40\x19\xad\x0c\x25\xb9\x66\x71\xff\x3f\xac\xa7\x80\x3d\xf3\x01\x8a\x70\xd5\xa0\x94\xca\x94\x89\xa6\xc2\x67\x90\x5e\x6d\x2b\xe9\xce\xa2\x90\xe7\xc1\x78\x08\x77\xdc\x00\x17\x20\xd8\x78\x32\x3e\x18\x30\x6a\xb0\xa4\xa4\xea\x0c\xa9\x26\xaf\xaf\x7b\xfe\x7b\xdc\xd5\xa8\x75\x8f\x28\xd8\xf8\xc4\xa9\x5f\x94\xc1\x64\xbe\x7e\xf8\xda\xc7\xf9\x7c\xbe\x47\xd6\x66\x2c\x62\x0b\xac\x95\x5e\x65\x70\x72\xa3\xa9\xb4\xb4\x42\xb8\x45\xe3\xe0\x76\x71\x72\x06\x0e\x8d\x4b\x1c\x59\x55\x04\x0d\xe1\x21\xb7\x24\xbc\x62\x93\xc1\x02\xb5\xe6\xd6\xbb\x40\xf4\x9e\xd9\x3b\x6f\xb1\x81\x0f\x2c\xe2\xc9\x82\x9b\x55\x4c\x03\xa6\xe9\x64\x92\x4c\xd3\xc9\x25\xdc\x2d\x95\xf7\x64\xcf\xe0\xb3\x11\xa3\x30\xf4\x45\x09\x32\x8e\x32\x58\x2c\xe0\xc6\x7b\xab\xf2\x36\x70\xc3\xf9\x28\x0d\x0e\x0d\xc6\xc3\x01\x0c\xb7\x16\xc5\xcf\xaf\xec\xe1\xa1\x55\x9e\x00\x35\x59\xef\xce\x20\x6f\x3d\x88\xd6\x79\xae\x01\x8d\x84\x8a\x74\x53\xb4\x1a\x0c\x7b\x72\x50\xb0\x85\x82\xf5\xbd\x03\x4b\x18\xe2\x02\x5f\x11\x48\x16\x6e\x14\xd8\xbe\xd1\x43\xab\x2c\x39\x40\xc8\xd1\x51\x24\xa8\x59\xaa\x42\x91\x05\xa1\xd1\xc5\xb1\xa8\x05\x16\x5c\xd7\x6c\xc0\xf9\x95\xee\x89\x43\x0a\x7e\xd5\x90\x8b\x81\xe6\x2e\x11\x9d\xd6\x18\x4d\x5f\x8f\x0c\xa6\x69\xe7\x77\x97\x41\xf7\xdd\xd5\x25\x67\x2b\xc9\x66\x30\x69\x9e\xc0\xb1\x56\x12\x8e\x89\xe8\xe5\x26\xe6\x95\x2c\x95\xf4\xd5\xa6\xd5\xfd\x8d\x45\xa9\x5a\x97\xc1\x79\x57\xb1\xed\xe5\xd5\x2c\xee\xef\x23\xf7\xdc\xf4\xe5\xec\x0f\x72\xf6\x9e\xeb\x4d\x39\xb7\x91\x4d\xa6\xd1\xf9\x44\x54\x4a\xcb\x6d\x8e\x35\x24\xdd\x05\x08\x96\x14\x07\xf7\xab\x8a\xcd\x0f\x2d\x20\x03\x6d\x03\xae\x41\x41\x90\x93\x5f\x12\x19\xa8\x5b\xed\x55\xa3\x09\x7a\xb2\x7f\x4c\x3c\x85\x5d\x4b\xb7\x9f\x94\x5c\xbc\xec\xf8\x8e\x56\x61\xa8\xce\x2e\x47\x22\xd1\x94\x64\xb7\x25\x46\x4b\xd7\xff\x87\xa0\xd9\x7c\x36\xdb\x79\xd5\x1a\xd4\xfb\xf8\xfa\xec\x12\xad\x09\xad\x3a\xb4\x01\xf1\xf2\x4d\x3a\x3f\x84\xda\x59\xb1\x7f\x58\x99\x82\x0f\xf2\x4f\xf2\x79\xfa\x96\xf6\x42\x76\xc8\x5f\x26\xc7\x43\xf8\x68\xe4\xe6\xc7\x82\xe1\xf8\x4f\x00\x00\x00\xff\xff\xe4\x31\x71\x04\x9b\x05\x00\x00"

func assetsStylesCssBytes() ([]byte, error) {
	return bindataRead(
		_assetsStylesCss,
		"assets/styles.css",
	)
}

func assetsStylesCss() (*asset, error) {
	bytes, err := assetsStylesCssBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/styles.css", size: 1435, mode: os.FileMode(436), modTime: time.Unix(1481510073, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesFactoidInfoHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x5c\x91\x41\x6b\xe3\x30\x10\x85\xef\xfe\x15\x83\xd9\xeb\xda\xe4\x1a\xb4\x62\x61\x61\xa1\x50\x7a\x68\xd3\x73\x19\xa4\x97\x5a\xd4\x96\x8d\xa4\x24\x84\x41\xff\xbd\x44\x56\x92\xb6\xbe\x78\xf4\xe6\x7b\xe8\xf1\x24\x62\xb1\x77\x1e\xd4\xc6\x74\x1e\x11\xdb\x9c\x1b\x55\x46\xdd\x34\xaa\xaf\x93\x08\xbc\xcd\xb9\xb9\xd3\x66\xf6\x09\x3e\x15\xdc\xba\x23\x99\x91\x63\xfc\x53\x64\x76\x1e\xa1\xbd\x98\x4e\x2e\x0d\xd4\xfd\x67\x93\x66\x67\x7f\x90\x0b\xbf\xe3\xf7\x00\xb6\x85\x25\x22\x52\xc3\x46\x57\x76\x4b\x22\x57\xdf\x13\x4f\xc8\x59\xf5\xc3\x46\x37\xaa\xb7\xee\xa8\x9b\x15\x5f\xf4\xe3\x6c\x3e\x50\x60\xb7\xa7\xee\x21\xae\xe7\x9c\xcf\x88\x22\x18\x23\x72\xf6\x73\xcd\xae\xfa\x45\xdf\x7c\x1c\x13\xc1\xba\xb4\xa5\xbf\x35\xe5\x21\x22\xd0\xaf\x6e\x07\x9e\xa8\xbb\xec\x5f\x23\x42\xce\x22\xdd\x7a\xff\xb5\x01\xaa\x9f\xf3\x24\x62\x06\xf6\x1e\xe3\x9b\xe7\x09\xdf\xcc\xff\xd6\xc5\x17\x9e\x13\x89\x04\x8c\xc9\x4d\x58\x99\x9d\x9b\x10\x13\x4f\x4b\xcd\x56\xc3\x05\x5c\x1b\x8a\xf3\x21\x18\xb4\x5a\x99\xd9\x42\x8b\x74\xcf\x7c\x7a\x29\xda\xc5\x51\x44\xd5\x2f\x01\xb7\x5a\xee\xaf\x54\xfe\x9f\x01\x00\x00\xff\xff\xf6\x34\xd0\xe9\xdb\x01\x00\x00"

func templatesFactoidInfoHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesFactoidInfoHtml,
		"templates/factoid-info.html",
	)
}

func templatesFactoidInfoHtml() (*asset, error) {
	bytes, err := templatesFactoidInfoHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/factoid-info.html", size: 475, mode: os.FileMode(436), modTime: time.Unix(1481510073, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesFactoidListHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x9c\x55\x4d\x6f\xdc\x36\x10\xbd\xeb\x57\x4c\x16\x45\xd0\x02\x2b\xa9\xf9\x38\x39\x8c\x90\x20\x40\x01\x03\x4e\x0f\xb5\x8d\xa2\xa7\x78\x56\x1a\x49\xac\xf9\xa1\x92\xd4\xae\x05\x42\xff\xbd\x20\x45\xad\xb5\x75\x72\xe9\x4d\x1a\xce\xcc\x7b\xf3\xe6\x89\xf2\xbe\xa1\x96\x2b\x82\x9d\x75\x93\x20\xbb\x9b\xe7\x8c\xc5\xc7\x2a\x1b\x45\xd1\x62\xed\x34\x6f\x72\xc1\xad\x03\x9f\x01\x00\x0c\xd8\x34\x5c\x75\x57\xf0\xeb\x87\x6c\x7e\x91\x23\x78\x4a\x0b\x6f\x79\x6c\x74\x05\x4a\x2b\xfa\x10\xa3\x12\x4d\xc7\x55\x2e\xa8\x75\x57\xf0\x76\x78\xba\x88\x1a\xde\xf5\x6b\x78\xce\x5e\xf4\x2d\x04\x5a\x97\x4b\xdd\xf0\x96\x53\x93\x60\x5a\xa1\xd1\x5d\x41\x2c\x0d\x55\xac\x4c\xe4\xbd\x27\xd5\xcc\x73\xf6\x3c\x60\xad\x95\x23\xe5\xe2\x84\x0d\x3f\x42\x2d\xd0\xda\x8f\x31\x8c\x5c\x91\xd9\x55\x17\xf1\x01\x3b\xca\x7b\xc2\x26\x9e\x04\x2c\xd6\xbf\xa9\x7e\x5b\x48\x59\x56\xf6\x6f\x2a\x66\x25\x0a\x51\x7d\x86\x13\x4e\xe0\x34\x58\xa7\x0d\x01\xaa\x06\x0c\x39\xc3\xe9\x48\xe0\xe8\xc9\xb1\x72\xc9\xcb\x58\xd9\xf0\xe3\x25\xca\xc1\xe6\x35\x0a\xa1\x47\x07\xcf\x8f\x39\x57\xad\x3e\x83\xbe\xaf\xee\x2d\x57\x1d\x6c\xa0\xdf\xa7\xb3\xa1\xba\xc5\x09\x58\xad\x1b\xaa\x5e\x25\xbd\x14\x4a\x62\x65\x0c\x01\x57\x80\x6a\x82\xba\x47\xa5\x48\xc0\xa9\x27\x43\xf0\x15\xcd\x91\x2b\xe0\x36\x50\x36\xa3\x02\x84\x54\x5a\xc0\x75\x0b\xae\xa7\xf5\x1d\x1c\x3e\x92\x05\x34\xdd\x28\x49\x39\xbb\x5f\x0c\x60\xf4\x91\x37\x14\x12\x25\x60\xeb\xc8\xc4\x9a\x80\xbb\x07\x3b\x60\x4d\xb9\xa5\x01\x0d\x3a\x6a\x0a\xb8\xeb\x69\x25\xb8\xb2\x62\x24\x2b\x39\x5a\xc7\x4a\x92\x15\x1c\x08\xd0\xc5\x0e\x47\x32\x13\x1c\xa8\xe3\x4a\x85\x79\x75\xe4\xb2\x18\x84\xac\xc5\x8e\x42\x37\x6e\x03\x73\x14\x56\x43\x50\x8b\x1a\x40\x38\xa0\xea\xf2\x75\x06\x56\x0e\x67\x75\xfe\xd2\x23\xd4\xa8\x96\xf4\x8b\x51\xe1\x30\x81\xc5\x29\xe0\x2c\xec\x3e\xc9\x45\x96\xf5\xbc\x23\x07\x2f\x15\x2d\xe0\xb3\x85\x03\xb5\xda\xd0\xfe\x2c\xc4\x59\x9f\x08\x8b\x16\x94\x36\x12\xc5\xc2\xe4\xff\xee\xfc\x4f\xc3\xdd\x0f\xb7\x7e\x6f\xa3\xfc\xff\xa1\x6e\x48\x92\x3c\x90\xd9\xf2\x86\xe4\xfa\x55\xfb\x5a\x4b\x19\x0c\x1a\xdc\x4a\x6e\xb3\xfa\xad\x6c\xd7\x2d\x4c\x7a\x04\x45\xd4\x5c\xd8\x21\xf5\x0a\xc5\xe9\xab\x01\x45\x27\xc1\x15\xd9\x3d\x34\x1a\x5c\xcf\xed\xd5\x32\xf5\x60\xa8\xfa\x01\x3b\x7a\x42\x39\x08\x82\xd7\x6f\xb2\x87\x87\x87\xec\xeb\x28\x1c\x0f\x3d\xb2\x2f\x4b\xfb\x10\x4d\x74\x59\x19\xfa\x7c\x8f\x96\x0c\x55\xa1\x8b\x5c\xcb\x61\x40\xe3\xec\x1e\xfe\x1e\xad\x03\x6c\x82\x2f\x5e\xbf\x4d\x8e\x39\x69\xf3\x68\xa1\xd5\x06\x50\x88\x55\x03\x1b\x3e\x8f\xf4\x31\xe4\x70\x18\x1d\x28\xed\x42\x52\xc4\xdb\x7a\xca\xee\xc3\x56\x5d\x4f\x13\xa0\x38\xe1\x64\x61\x4c\x0b\x20\xe5\xb8\xa1\xd5\x9f\xaf\xb6\x1b\xcf\x58\xff\x6e\xbd\x2e\xe0\x86\x07\xbb\xf7\xef\xaa\x8c\x8d\x62\xf5\xc1\xf6\x82\xdb\x85\x1b\x0b\x0c\xaa\x8e\xa0\x08\xd9\x30\xcf\x19\x00\x13\xbc\x62\x08\xbd\xa1\xf6\xe3\xae\x5c\xe9\x94\xde\xf3\x16\xe8\x1f\x28\x6e\x6b\x3d\xd0\x97\xf4\x79\xef\x76\xf3\xfc\xcd\x7b\x12\x96\xe6\xd9\xfb\x8b\xc3\x10\x88\x17\x62\xe9\x7d\x91\x58\xfd\x8e\x92\xe6\x39\x79\x0e\x92\x99\xbc\x87\xed\x39\xcc\x73\xda\x45\xc4\x54\xf4\x1d\x4c\x66\x07\x54\xeb\x50\xdc\xe6\xe9\xba\xc9\xb5\x12\xd3\xae\x7a\x41\x84\x95\x21\xbf\x4a\x7c\x58\x89\x2b\x81\x88\x50\x5c\xdb\x1b\x5d\x3f\x52\x33\xcf\x3f\x8b\xf8\xf0\xcb\x7a\x95\x27\x9a\x5b\xb4\x8b\x5f\xc2\xae\x0a\xaf\x70\xfe\x43\x1c\x26\xf8\xe4\xfd\x89\xbb\x3e\xec\xcb\xc0\x4f\xc5\x1d\xa1\x84\xe2\x06\xad\xbb\xb7\x64\xa2\x48\x8b\x08\x97\x10\x10\x9c\xe1\x7d\x9a\xe3\x5b\xfc\x8e\xb6\xc5\xe7\x51\xc0\x7b\x43\xc2\x71\x49\xcb\xc1\x1d\x97\x64\x1d\xca\xe1\x3c\xe5\x4a\x7a\x30\xb4\x72\xb6\x7a\x34\x35\xed\xaa\x55\xef\xe2\x0f\x3c\xdd\xc6\xd8\x59\xeb\xb3\xef\x59\x29\x78\x74\x06\xa9\x26\x38\x82\x95\xe3\xf3\x7f\x24\x91\xfe\x37\x00\x00\xff\xff\x43\xa3\x15\x02\xc3\x07\x00\x00"

func templatesFactoidListHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesFactoidListHtml,
		"templates/factoid-list.html",
	)
}

func templatesFactoidListHtml() (*asset, error) {
	bytes, err := templatesFactoidListHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/factoid-list.html", size: 1987, mode: os.FileMode(436), modTime: time.Unix(1481510073, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesHomeHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x74\x92\x41\x6b\x1b\x31\x10\x85\xef\xf9\x15\xd3\x3d\xa7\x56\x9d\x63\x51\x04\xa1\x50\x1a\xe8\x42\x20\xa1\xd7\x30\x96\x66\x57\x13\x6b\xa5\x45\xd2\xae\x9b\x1a\xff\xf7\x22\xad\xed\x24\x04\xdf\x84\x66\xf4\xbd\x79\x4f\xb3\xdf\x1b\xea\xd8\x13\x34\x3a\xf8\x4c\x3e\x37\x87\xc3\x95\x34\x3c\x83\x76\x98\xd2\x6d\xbd\x46\xf6\x14\x1b\x75\x05\x00\xf0\xbe\x36\x62\x4f\x5f\x2d\xa1\x39\x57\x6b\x87\x5d\xab\x5f\x61\x20\x29\xec\x5a\xc9\x34\xa0\x73\x8a\x13\xec\x2c\x45\x82\x6c\x09\x2c\x61\xcc\xc0\x49\x8a\xa5\xb8\x70\x85\xe1\x59\x5d\x2d\xe7\x51\x3d\x59\x4e\xc0\xa9\xf6\xef\x68\x03\xec\x33\xc5\x0e\x35\x41\x17\x22\xb4\x18\x67\xf6\x2b\x29\xc6\xe3\x63\x7b\xa3\x24\x82\x8d\xd4\xdd\x36\xa2\x43\x9d\x03\x9b\xd4\xa8\x9f\xc7\x93\x14\xa8\xa4\xb0\x37\xea\x84\xff\xc3\xb4\x83\x14\xa6\xa8\xe9\x1a\x2c\xa7\x1c\xe2\xeb\x35\xa0\x37\x80\x93\xe1\x0c\x2e\xf4\xa9\x2a\x9d\x58\x97\xb4\x4a\x63\xa3\x7e\x87\xfe\x82\x46\x70\x66\x81\x85\xae\x7a\x79\x74\xa8\xb7\xa0\x2d\x7a\x4f\x2e\xc1\x88\x29\xd7\xfb\xf5\xb7\x2d\x0c\x94\x12\xf6\x04\x8e\x07\xce\x2b\xb9\x89\x6f\x99\x3e\x05\x98\x0b\x6e\x8c\x3c\x63\xa6\x33\xe0\x1a\x5e\xc3\x04\xc3\x94\x32\x6c\x08\x12\xf7\x9e\x0c\xb0\x87\x1d\x67\xbb\x68\x5d\x1a\x9c\xfd\xcc\x99\x52\xa3\x1e\x8e\xc8\x1f\x0b\x12\xee\x4b\x01\x33\x07\xff\xd9\xd2\x23\x39\xd2\xb9\xe6\xf4\x12\xd8\x43\x33\x4e\x1b\xc7\xba\xf9\x34\x17\x94\xcf\xfe\xf2\x41\xba\x0d\xe5\xfb\x03\xe8\x65\x35\x4e\xd0\xc9\xbd\xdb\x1c\xc7\x25\xca\xb3\x81\x7b\x9f\x23\x96\x37\x3d\xe5\x63\x72\x0f\x6d\xdd\x24\x5f\x7d\x5b\x9c\x09\x10\x74\x88\x91\x74\x99\x58\x0a\xc7\x1f\x71\x67\xc3\x36\xe7\xf1\xbb\x10\x3b\xde\xf2\xaa\xba\x08\x11\x07\x5c\xe9\x30\xd4\x3b\xd1\xe2\x5f\x1e\xf8\x1f\x3d\xdf\x75\x1d\xc7\x81\xcc\x73\x8b\x2f\x21\x72\xe6\x12\x52\x7b\xd7\x96\x30\x60\x0c\xce\xb1\xef\xdf\x74\xa4\x28\x06\x8e\xcb\xbb\xdf\x93\x37\x87\xc3\xff\x00\x00\x00\xff\xff\x76\xc3\x87\x61\x54\x03\x00\x00"

func templatesHomeHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesHomeHtml,
		"templates/home.html",
	)
}

func templatesHomeHtml() (*asset, error) {
	bytes, err := templatesHomeHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/home.html", size: 852, mode: os.FileMode(436), modTime: time.Unix(1482217354, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesInviteListHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xb4\x57\xef\x6e\xdb\x36\x10\xff\xee\xa7\xb8\x0a\x01\x24\x63\x91\xdc\xac\xfb\x30\xd8\xb2\xb0\x2e\x6d\xd1\x0c\x59\x57\x34\x29\x36\x60\xd8\x07\x46\x3c\x49\x8c\x29\x52\x20\x69\xa7\x99\x2a\x60\x4f\xb3\x07\xdb\x93\x0c\xa4\xfe\x66\x89\x53\x17\x5d\xf3\x21\xb6\x8f\xf7\xe7\x77\x77\xbf\x3b\x51\x75\x4d\x31\x63\x02\xc1\xd3\xe6\x96\xa3\xf6\x9a\x66\x16\xbb\xaf\xc9\x0c\x00\x20\xba\x96\x4c\x84\x57\x46\x40\xed\x7e\xdb\xbf\x4a\x6a\x66\x98\x14\x4b\x20\x57\x5a\xf2\xad\xc1\xd5\x70\x66\x64\xb5\x84\x13\x85\xe5\x28\x52\x2c\x2f\xcc\x12\x9e\x4e\x94\xf0\x83\x09\x8d\x22\x42\x67\x52\x95\x4b\xe0\xf2\x06\x55\x4a\x74\xe7\xa7\x69\x23\x97\x61\x5a\x10\x21\x90\x43\xf1\x6c\x12\x3d\x93\xc2\x84\x9a\xfd\x89\x4b\xf8\xf6\xbb\xea\xc3\x1e\x93\xa8\x44\x43\xee\x59\xd9\xbc\x96\xc0\x0c\xe1\x2c\xdd\x6f\xa8\x35\xc9\xf1\xe1\x88\x27\xdf\x8f\x11\xe3\x45\x57\xa7\xba\x46\x41\x9b\x66\x36\xd6\x32\x95\xc2\xa0\x30\xae\x98\x94\xed\x20\xe5\x44\xeb\xb5\x13\x13\x26\x50\x79\xc9\x1d\x79\x45\x72\x0c\x0b\x24\xd4\x9d\x58\xef\x71\x71\x92\xbc\x55\x6c\x47\x0c\xc2\x69\x8b\x4c\xc7\x8b\xe2\x24\x89\x75\x49\x38\x4f\x7e\x92\x4c\xc0\xb6\x82\x02\x15\x3e\x89\x17\xad\x70\x16\x2f\x28\xdb\x25\xb3\x59\xac\x31\xb5\x0d\x6a\x7d\xd5\x35\x20\x49\x0b\x88\x7a\x47\xd0\xb4\x59\x4f\x21\x8c\x05\x48\x89\xa2\xee\x5f\x78\xc5\x65\xba\xf1\x80\x12\x43\x42\x46\xd7\x5e\x5d\x47\x67\x2f\x9a\xa6\x83\xd8\xc2\x7c\x96\xd4\x75\xf4\x86\x94\xd8\x34\xf1\xa2\x78\x36\x39\xd2\x15\x11\xbd\xf7\x9e\x44\x13\xd3\xba\x06\x96\xc1\x51\xf4\x46\x9a\x73\x99\xe7\x48\xcf\x04\x84\x1d\xb0\xc1\x07\xe9\x1d\x58\xdb\x0e\x88\x30\xa8\x48\x6a\xd6\xde\x89\x97\x9c\xcb\x1c\xce\x04\xbc\x62\x4a\x9b\x78\x41\xa6\xde\x43\x40\xae\xf1\x8e\x4b\x2b\x64\x19\x10\x41\x21\xba\x30\xc4\x6c\xf5\xb9\x24\x14\x29\x44\xcf\x77\x84\x71\x72\xc5\xf1\x71\x08\x70\x65\x44\x58\x29\x56\x12\x75\xfb\x10\x9c\x98\xf5\xca\x19\x81\x8c\x84\x9a\x93\x74\xe3\x25\xf1\x82\x25\xe0\x3a\xd6\x75\xe0\x60\xa8\x77\x61\x1e\x8c\x0d\x28\xd3\x36\x1b\xea\x25\xcf\xb9\x42\x42\x6f\x5d\x78\xa4\x9f\x0e\xfc\xb0\xdf\x1b\xa2\x04\x13\xb9\x97\x9c\x16\x98\x6e\xe0\x15\x61\xdc\xe2\x81\xc7\x12\x3e\xe5\x2c\xdd\x80\x91\x2e\xf4\xfd\xc0\x82\xde\x4b\x78\x9f\x6c\x22\x8a\x17\x96\x56\x13\x96\x55\x03\x81\xd1\x10\x2f\xf9\xa1\xae\xa3\xf7\x1a\x55\xcb\x48\x48\x15\x12\x83\x14\x98\xd8\x31\x83\x20\x05\xd4\xb5\x42\x6e\x58\x89\x10\x5d\xb2\x12\xb5\x21\x65\x65\xa9\x5b\x3d\xec\xd3\xad\x02\xcf\x72\xfc\x12\x3f\x98\x89\x62\x37\x6a\xfd\x7c\xb5\x28\xe3\xc5\x30\x78\xdd\xb9\x1b\x31\x3b\x3b\x15\x27\x29\x16\x92\x53\x54\xda\x03\xb7\x38\xd6\x1e\x65\xba\xe2\xe4\x76\x29\xa4\xc0\x7e\xf0\x1f\x30\x08\xdd\xf8\x90\x9e\xa3\xd3\xf9\xfb\xaa\xec\x9c\xe4\xb8\x1f\x16\x97\x39\x13\xfb\x20\x1d\x3c\xb3\x07\x85\xaa\x50\x50\xc7\xc2\x3d\xf9\x8f\xac\xbf\x97\x6a\xc5\x44\xff\xe9\xd6\xaf\x4b\xfa\x57\xa9\x36\x4c\xe4\xff\xfc\xf5\xf7\xe7\xe1\xc8\x98\x60\xba\xb0\x71\x1e\x69\x04\x13\x99\xfc\x74\xcd\x27\x23\x79\x50\xe8\xad\xd8\x08\x79\xb3\xb7\xde\xff\xf3\xb0\x1e\x04\x09\x95\x92\xea\x51\x40\x94\x88\x1c\xd5\x43\x5c\x78\x69\x6d\x21\x48\x1d\xd0\x54\x0a\x2d\x39\xce\x3f\x2f\x3c\x69\x17\xdc\xb5\x2b\xe6\x21\xa3\xf1\xe9\xe5\xd8\xcf\x6e\xf7\xa1\x53\xc5\x2a\x93\xcc\x76\x44\x41\xaa\x55\x76\x29\x37\x28\x60\x0d\x75\x7d\xad\x21\x3a\xbd\x78\xf7\xaa\x69\x56\xee\x54\x1b\xa2\xec\xd3\x8c\x89\xf7\xef\xce\x07\x8d\x73\x72\x2b\xb7\xc6\xae\x72\x65\x2e\x6c\xcd\xed\xa1\x97\x2b\xb9\xad\xf4\xd2\x02\xf0\x7a\x7b\x81\x48\x9d\xf9\x60\xfb\x06\x91\xbe\x45\x55\x32\xad\x99\x14\x4d\x03\x1f\x3f\x42\xf0\xa4\x3b\x1b\x9f\x9c\x4d\x33\x5f\xcd\xb2\xad\x70\xcb\x07\xae\xb6\xc6\x48\xe1\x5a\x1a\xe0\xce\xcc\xbb\xab\x8c\x43\x58\x11\xf1\x92\xc3\x1a\x70\x67\x22\x8a\x1c\x73\x62\xf0\x92\xa8\x1c\xcd\x6a\x50\xa2\x6c\xe7\x74\x5a\xe5\xa8\x22\x0a\x85\x79\xc9\xb1\x44\x31\xd1\xea\xee\x0b\x67\x2f\x60\xed\x2c\xce\x23\xdb\x5f\x8d\x26\x62\x74\x35\x73\x6a\x2c\x83\x40\xdc\xc9\x60\x3e\xb9\x56\x51\x99\x6e\xad\xcb\x88\xcb\x94\x38\xe0\xeb\xbb\x15\x9c\x5c\x21\xd1\x6c\x95\x98\xde\xd8\x2c\x82\x23\xe2\x60\x1e\x05\x2d\xd0\x79\x94\x31\x41\x03\x9f\xf8\xf3\xd5\x10\xfe\x89\x55\xfa\xfd\xe9\x1f\x23\xb8\x8e\x7f\x53\x24\x77\xdd\x0f\xfe\xdb\x1b\x99\x86\x35\x08\xbc\x81\xd7\xed\xaf\xa0\x73\xde\x1d\x46\x1a\x4d\xe0\xff\x16\x5a\x1a\x84\x8e\x19\xfe\xf1\xc8\x92\xf9\x58\x2e\x59\xd9\x14\xad\xb3\x31\x6e\x89\xa6\x90\x74\x09\xfe\xdb\x5f\x2e\x2e\xfd\xe3\x41\x9e\x2a\xa4\x28\x0c\x23\x5c\x2f\xc1\xd7\xa4\xc4\x50\x2a\x96\x33\x31\xd1\xe9\xe2\x2f\xfb\x2f\x2d\xf8\xd5\x88\x7e\xab\x6c\x71\xfc\x45\xfb\xec\xd3\x0b\x1f\xbe\x19\x7b\xd6\xe9\x75\x1d\x76\x4b\xf1\xf5\xe5\xcf\x96\xb3\x43\x57\x72\xec\x7b\xfe\xe3\xed\x19\x0d\xfc\x7d\xeb\xd8\x9f\x8f\xf6\x6d\xba\x19\x9a\xb4\x08\xb6\x8a\x1f\xf7\x59\xcf\x23\x53\xa0\x08\x7a\x86\x06\x0a\x75\x75\xbf\x01\x60\xc5\xd1\xb5\x96\xa2\x2f\x72\xf3\x5f\x43\x7b\x38\x35\x74\x2d\xb6\xc2\x48\x6e\xa6\xf2\x2f\x4e\xce\x6d\xb5\xfb\xa9\x0d\x1d\x6a\x77\x55\xc4\x65\x1e\x78\x6e\x8b\x2d\xbd\x63\x70\xe8\xee\x2a\x4e\xa9\x35\xb2\xf7\x8b\xe1\xf5\x8f\xa0\xfb\x08\x9b\x63\x18\xaa\x85\x4a\x4d\x8b\xf2\xb5\x0a\x32\x2d\x86\x0d\xd9\xf7\x6e\x35\x6b\x66\x47\x81\x3f\xbc\x42\xfa\xf3\x48\x8a\xc0\x4f\xed\x62\xf2\x8f\xa7\x6b\x6a\xbe\xb2\x77\xa7\x6e\xcf\x76\xaf\x53\xff\x06\x00\x00\xff\xff\x33\x9a\x62\xb6\x9a\x0e\x00\x00"

func templatesInviteListHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesInviteListHtml,
		"templates/invite-list.html",
	)
}

func templatesInviteListHtml() (*asset, error) {
	bytes, err := templatesInviteListHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/invite-list.html", size: 3738, mode: os.FileMode(436), modTime: time.Unix(1482220435, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesLogsIndexHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xbc\x55\x4d\x6f\xe3\x36\x10\xbd\xfb\x57\x0c\xb4\x28\x72\xa9\x24\xdb\x59\xa4\x80\x56\x11\x8a\xee\x1e\x5a\x20\x5b\x2c\xba\x6d\x81\x1e\x47\xe2\x48\x62\x43\x91\x02\x49\xc9\x31\x08\xfd\xf7\x82\xfa\x88\x15\xd7\x41\x90\xa0\x5d\xfb\x42\xcd\x90\xef\xbd\x79\x33\x94\x9c\x63\x54\x72\x49\x10\x18\x7b\x14\x64\x82\x61\xd8\xa4\xe3\x32\xdb\x44\x45\x8d\x52\x92\x08\x73\xf5\x10\x16\x4a\x5a\xe4\x92\x34\xb8\x0d\x00\x00\xe3\xa6\x15\x78\x4c\xa0\x14\xf4\xf0\x61\x0c\xf9\x55\x58\x0a\x75\x48\x40\xab\x03\x1c\x34\xb6\x53\xe2\xef\xce\x58\x5e\x1e\x47\x0c\x92\x76\x3a\x13\x1a\x8b\xda\x7e\xd8\x0c\xcf\xf1\x64\xc0\x78\x3f\xb3\x35\xa8\x2b\x2e\x13\xb8\x6e\x67\xae\x29\x10\x6a\x5e\xd5\x36\x81\xdd\xd6\xc7\x5f\x40\x5a\xe3\xe5\x58\xdc\x57\x5a\x75\x92\x85\x85\x12\x4a\x27\xf0\xae\xdc\xfa\xff\x84\x9e\x2b\xcd\x48\x87\x1a\x19\xef\x4c\x02\x37\x0b\x6b\x8b\x8c\x71\x59\x25\xb0\x8d\x76\xed\xc3\x14\x7f\x81\x15\xcf\xfd\xca\x85\x2a\xee\x27\xb8\x03\x67\xb6\xf6\xea\xb7\xdf\x4d\x81\x45\xcb\xcd\xfe\xfa\x3a\xff\xe1\xad\x95\x46\x52\x85\x0d\xea\x9e\xcb\x99\x5b\xb5\x58\x70\x7b\xf4\xb2\x6f\x9e\x12\xed\xf7\xfb\x57\xa0\x2d\xb5\xbc\xf1\x74\xd2\x73\xc3\x2d\xb1\xcb\x28\x69\x3c\x8f\x9d\x73\x24\xd9\x30\x6c\x4e\xa3\x39\xcf\xcd\x38\x9b\xde\xd3\x42\xa0\x31\xb7\xc1\x23\x55\x90\x8d\x80\xeb\x5c\x8b\x15\x85\x35\x21\xf3\xd9\xb4\xde\x65\x1f\x27\x8d\x70\xa7\x2a\x93\xc6\xf5\x2e\x4b\x4d\x83\x42\x64\x3f\x51\x81\x9d\x21\xf8\x2a\xb0\xb8\x87\x56\x53\xc3\xbb\x06\xb8\x81\x9c\x0b\x41\x0c\x5a\xd2\xd0\x19\xd2\x69\x3c\xed\x4f\x63\xc6\xfb\x6c\x33\x11\xd6\xfb\xec\x4b\x97\x0b\x5e\xc0\x0c\xef\xa1\xf7\xff\x56\x73\xd1\xa0\x59\xb5\xff\x39\xa7\x51\x56\x04\xd1\x02\x33\x0c\x8f\xb9\x11\x67\xb4\xe6\x36\x18\xc7\x32\x81\xd0\x39\x41\x12\xa2\xcf\xd4\xe4\xa4\xcd\x30\xac\xa0\x96\x23\x67\x91\xfa\x7d\x96\x22\xd4\x9a\xca\xdb\x20\x16\xaa\x32\xb1\x73\xd1\x2f\x9f\xfc\xd1\x77\xce\x45\xbf\x62\x43\xc3\x90\xc6\x98\xa5\x71\xfd\xfe\xec\x6c\x9b\x39\x17\x7d\xe9\x74\xab\x0c\x45\x7f\xa2\xe8\xc6\xad\xed\xd9\xae\xf8\x09\xe9\xd9\xe3\xd2\xd3\x55\xea\xe4\xa0\xe6\x3d\x5a\xba\x60\xa1\x73\xc0\x4b\x88\xfe\x52\x9d\x9e\x37\x2d\x7b\x20\x5c\xc0\x5e\xe5\x72\x08\xb3\xcf\x97\x20\x5f\xeb\xf9\x42\xeb\x1c\x2f\x41\x2a\x0b\xd1\xcf\x68\x3e\x8f\xb3\x3e\x0c\xab\x2b\x38\x95\xfe\xbf\xb6\xe8\xa2\x84\xb4\xcd\x52\x6a\x32\x3f\xf0\x80\x9a\xc6\x3c\xf6\xc8\x05\xe6\x82\xa0\x54\x1a\x6c\xcd\x0d\xcc\xae\x41\x3e\x5f\x04\x6e\x81\x29\x32\xf2\xca\x42\x8d\x3d\xc1\x8f\x53\x1d\x51\x1a\x53\x93\xf9\xb6\xaf\x7b\xf9\xdf\xce\x08\x90\x64\xa7\xce\x9e\xb2\x3e\x23\x0c\xad\x3b\xe4\x7b\xe9\x67\xe3\x0e\x8f\xaa\xb3\xd1\xa7\x8f\x9d\xd6\x24\xed\x1f\x86\x74\x34\x5e\x65\xbf\x7a\xc4\x9a\x25\xfe\xae\xa0\xe7\x74\x00\x5b\x13\x78\x83\x47\x13\xda\x79\xfc\x66\x1f\xcc\xf7\xb0\x74\xe1\xca\xb9\x05\xff\xab\xff\x56\x4d\xb8\xbf\xdd\x41\xe0\x3f\x1d\xad\x49\x34\x21\x0b\x86\xe1\x2a\xab\x34\x4a\x0b\x58\x14\x64\x8c\xef\x50\xf4\xa4\x74\xaf\x75\xd4\xff\xad\xe4\x18\x5e\x49\xe0\xf2\x19\x25\x2b\x8f\x97\x67\xff\x62\x9d\xec\x9e\x9b\xfb\x4f\x00\x00\x00\xff\xff\xfc\x57\x79\x34\x17\x08\x00\x00"

func templatesLogsIndexHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesLogsIndexHtml,
		"templates/logs-index.html",
	)
}

func templatesLogsIndexHtml() (*asset, error) {
	bytes, err := templatesLogsIndexHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/logs-index.html", size: 2071, mode: os.FileMode(436), modTime: time.Unix(1482197067, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesUserManageHtml = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xbc\x54\x4f\x6b\xdc\x3e\x10\xbd\xe7\x53\x0c\xfe\x1d\x72\xf9\x59\x26\x21\xa7\xa0\x88\x42\x72\x68\x61\xb7\x2c\x4d\x5b\xe8\x71\xd6\x1e\xdb\xa2\xb2\x64\x24\xd9\x6d\x10\xfa\xee\x45\xfe\xb3\xbb\xd9\xee\x25\x50\xb2\xa7\x95\x66\xf4\xe6\xcd\x7b\x33\x0e\xa1\xa2\x5a\x6a\x82\xac\x34\xda\x93\xf6\x59\x8c\x57\xbc\x92\x23\x94\x0a\x9d\x7b\x98\xae\x51\x6a\xb2\x99\xb8\x02\x00\x38\x8d\xf5\xd8\x50\xde\x12\x56\x29\xca\xdb\x1b\xf1\xcd\x91\x85\x9d\x35\xb5\x54\xc4\x8b\xf6\x46\x70\xd7\xa1\x52\x62\x8b\x1a\x1b\x82\xd2\x68\x4d\xa5\x97\x46\x3b\x40\x5d\x81\x32\x0d\x98\xc1\xf3\x62\xce\xe2\x45\x25\x47\x71\x35\x97\x69\x6f\xc5\xb3\xc2\xf2\x27\x2f\xda\xdb\xbf\x2b\x97\x2d\x6a\x4d\x2a\xdf\x9b\xdf\xf9\x39\xc3\xf4\x0b\xc1\xa2\x6e\x08\xd8\xe3\x9c\xe8\x62\x3c\xc4\x26\x1c\xe7\x5f\x14\x3d\x64\xc6\x56\x64\xef\x21\x0f\x41\x91\x06\xb6\xa5\x6e\x4f\xd6\xc5\x78\x02\xb5\x3e\x79\x7d\x33\x73\xbc\x13\x1c\xa1\xb5\x54\x3f\x64\x85\x32\x8d\x2b\x42\x60\x9f\x9e\xd2\xf3\xff\x42\x60\x9f\xb1\xa3\x18\x79\x81\x82\x17\xed\xdd\x85\xf7\xbd\x08\x81\xed\x06\xdb\x1b\x47\xec\x3b\xaa\x61\x4a\xef\xcf\x6a\x17\xaf\x8a\x9f\x1d\x43\x20\x5d\x2d\xcd\x9d\xeb\xb7\xb3\x72\x44\x4f\xb0\x6a\x70\x94\x32\x04\x90\x35\xb0\x1f\x66\xb0\x4b\xd2\x9a\x03\xf9\x0a\xf6\x26\xb5\x73\x58\xf4\xbe\x04\xf9\x56\xed\xd7\xb2\x21\xc8\x1a\xb4\xf1\xc0\x3e\xa2\xdb\xa2\x1d\xa5\x8e\x51\x9b\xbc\x9b\xfe\xae\xad\xbf\x8b\x55\x17\xa9\xf0\x5e\x70\xea\xc4\xc6\x34\x0e\xd0\xd2\x14\xc7\x11\xa5\xc2\xbd\x22\xa8\x8d\x05\xdf\x4a\x07\x8b\x7a\xb0\xa7\x12\x07\x47\x20\x3d\x54\x86\x9c\xbe\xf6\xd0\xe2\x48\xf0\x61\xee\x87\xf1\x82\x3a\x91\xec\x3f\xf5\xf4\xdf\xcf\x0b\x90\xae\x8e\x2e\x1f\xa3\x29\xa2\x1c\xad\x6e\x25\x4f\xd3\x8c\x6c\xf0\xc5\x0c\x9e\x3d\x3d\x0e\xd6\x92\xf6\x69\xc3\xd9\xb4\x98\xd3\xae\x1f\x70\x7a\xf1\xd5\xc0\x28\xe9\x17\xf8\x96\xd2\x62\xbb\x49\x80\x7e\x19\xc1\x45\x03\xf7\x3f\xac\x2e\x5c\x87\xb0\x62\x3f\x7b\xb4\x7e\xc6\xfc\xb2\x81\xac\xb1\x66\xe8\xdd\xbd\x25\xac\xb2\x18\xaf\x45\x63\x51\x7b\xc0\xb2\x24\xe7\x92\x43\xec\xd0\x72\xe2\x38\x71\x7e\x0f\x1a\x4e\x36\x1a\xa4\xbe\xc0\xe0\x44\xcf\xf5\x9c\x3e\xa2\xb3\xb4\x8b\x99\x7f\x02\x00\x00\xff\xff\x5d\xf8\x1a\x40\x68\x05\x00\x00"

func templatesUserManageHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesUserManageHtml,
		"templates/user-manage.html",
	)
}

func templatesUserManageHtml() (*asset, error) {
	bytes, err := templatesUserManageHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/user-manage.html", size: 1384, mode: os.FileMode(436), modTime: time.Unix(1482197067, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"layout.html":                 layoutHtml,
	"assets/styles.css":           assetsStylesCss,
	"templates/factoid-info.html": templatesFactoidInfoHtml,
	"templates/factoid-list.html": templatesFactoidListHtml,
	"templates/home.html":         templatesHomeHtml,
	"templates/invite-list.html":  templatesInviteListHtml,
	"templates/logs-index.html":   templatesLogsIndexHtml,
	"templates/user-manage.html":  templatesUserManageHtml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"assets": &bintree{nil, map[string]*bintree{
		"styles.css": &bintree{assetsStylesCss, map[string]*bintree{}},
	}},
	"layout.html": &bintree{layoutHtml, map[string]*bintree{}},
	"templates": &bintree{nil, map[string]*bintree{
		"factoid-info.html": &bintree{templatesFactoidInfoHtml, map[string]*bintree{}},
		"factoid-list.html": &bintree{templatesFactoidListHtml, map[string]*bintree{}},
		"home.html":         &bintree{templatesHomeHtml, map[string]*bintree{}},
		"invite-list.html":  &bintree{templatesInviteListHtml, map[string]*bintree{}},
		"logs-index.html":   &bintree{templatesLogsIndexHtml, map[string]*bintree{}},
		"user-manage.html":  &bintree{templatesUserManageHtml, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
