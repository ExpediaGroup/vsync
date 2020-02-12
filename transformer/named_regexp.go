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

package transformer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/ExpediaGroup/vsync/apperr"
)

var ErrRegexParse = errors.New("regex parse error")

// NamedRegexp is a regexp type but has extra functions to find the named matching substrings
type NamedRegexp struct {
	Plain string
	*regexp.Regexp
}

// FindStringSubmatchMap returns the map{name: foundString} from regexp
func (r *NamedRegexp) FindStringSubmatchMap(s string) map[string]string {
	matchMap := map[string]string{}

	matches := r.FindStringSubmatch(s)
	matchNames := r.SubexpNames()

	if len(matches) == 0 {
		return matchMap
	}

	for i, name := range matchNames {
		if name == "" {
			continue
		}
		matchMap[name] = matches[i]
	}

	return matchMap
}

type NamedRegexpTransformer struct {
	Name string
	From NamedRegexp
	To   string
}

func NewNamedRegexpTransformer(name string, from string, to string) (NamedRegexpTransformer, error) {
	const op = apperr.Op("transformer.NewNamedRegexpTransfomer")

	t := NamedRegexpTransformer{}

	r, err := regexp.Compile(from)
	if err != nil {
		log.Debug().Err(err).Str("from", from).Msg("cannot parse the regular expression")
		return t, apperr.New(fmt.Sprintf("From regular expression %q", from), err, op, ErrRegexParse)
	}

	t.Name = name
	t.From = NamedRegexp{
		Plain:  from,
		Regexp: r,
	}
	t.To = to

	return t, nil
}

func (t NamedRegexpTransformer) Transform(path string) (string, bool) {
	matchMap := t.From.FindStringSubmatchMap(path)

	tStrs := []string{}
	toNames := strings.Split(t.To, "/")

	if len(matchMap) == 0 {
		return "", false
	}

	// matchMap result could be map[app:secrets env:test mount:rockcut team:]
	// if any there are any unmatched key like team in above example then we return false
	for _, v := range matchMap {
		if v == "" {
			return "", false
		}
	}

	// example of transformer To string is mount/env/app/team
	// but it can have non group names like mount/env/app/team/v1 where v1 is not present in regexp itself as a group name
	//   then we append v1 as a string instead of taking the actual value from regexp matchMap
	for _, toName := range toNames {
		v, ok := matchMap[toName]
		if ok {
			tStrs = append(tStrs, v)
		} else {
			tStrs = append(tStrs, toName)
		}
	}

	tStr := strings.Join(tStrs, "/")
	tStr = regexp.MustCompile("/+").ReplaceAllString(tStr, "/")

	log.Debug().
		Str("name", t.Name).
		Str("before", path).
		Str("after", tStr).
		Interface("matchMap", matchMap).
		Strs("toNames", toNames).
		Msg("transformed")
	return tStr, true
}
