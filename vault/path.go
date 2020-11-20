// Copyright 2019 Expedia, Inc.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vault

import (
	"regexp"
	"strings"

	"github.com/ExpediaGroup/vsync/transformer"
)

var rMount string = `(?P<mount>\w+\/?)`          // mount required word
var rMeta string = `(?P<meta>((meta)?data)\/?)?` // meta optional word from the list
var rRest string = `(?P<rest>.*)?`               // env optional word from the list
var rPath = transformer.NamedRegexp{
	Plain:  rMount + rMeta + rRest,
	Regexp: regexp.MustCompile(rMount + rMeta + rRest),
}

// GetMetaString will place metadata in the string as the second word after mount
// secret -> secret/metadata
// secret/ -> secret/metadata
// /secret/metadata -> secret/metadata
// secret/metadata/ -> secret/metadata
// secret/platform -> secret/metadata/platform
func GetMetaPath(s string) string {
	matches := rPath.FindStringSubmatchMap(s)

	if matches["meta"] == "" {
		matches["meta"] = "metadata/"
	}

	p := matches["mount"] + "/" + matches["meta"] + matches["rest"]
	p = regexp.MustCompile("/+").ReplaceAllString(p, "/")

	// if p[len(p)-1:] == "/" {
	// 	p = p[:len(p)-1]
	// }
	return p
}

// GetDataPath is similar to GetMetaPath but replaces first metadata with data
func GetDataPath(p string) string {
	return strings.Replace(GetMetaPath(p), "/metadata", "/data", 1)
}

// GetMount returns the first portion of path assuming its the mount
func GetMount(p string) string {
	return p[:strings.Index(p, "/")+1]
}

// ParentPath return the parent of key path by removing the part after last /
func ParentPath(s string) string {
	return s[:strings.LastIndex(s, "/")]
}
