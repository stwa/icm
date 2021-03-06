// Copyright © 2018 Marcel Meyer meyermarcel@posteo.de
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package input

import (
	"strings"
	"unicode/utf8"
)

// Validate validates inputs. Each input is validated and values are assigned.
func Validate(in string, newInputs []func() Input) ([]Input, error) {

	previousValues := make([]string, 0)
	inputs := make([]Input, 0)
	var err error
	for _, newInput := range newInputs {
		input := newInput()
		input.previousValues = previousValues

		matchIndex := input.matchIndex(in)
		if matchIndex != nil {
			matchPart := in[matchIndex[0]:matchIndex[1]]
			if input.toUpper {
				matchPart = strings.ToUpper(matchPart)
			}
			input.value = matchPart
			in = in[matchIndex[1]:]
		}

		previousValues = append([]string{input.value}, previousValues...)
		input.validateValue()

		inputs = append(inputs, input)

		if err == nil {
			err = input.err
		}
	}
	return inputs, err
}

// Input is a structured part of an input string.
type Input struct {
	runeCount      int
	matchIndex     func(in string) []int
	validate       func(value string, previousValues []string) (error, []Info, []Datum)
	toUpper        bool
	value          string
	previousValues []string
	err            error
	infos          []Info
	data           []Datum
}

// SetToUpper converts the matched value to upper case.
func (i *Input) SetToUpper() {
	i.toUpper = true
}

// NewInput returns a new Input.
func NewInput(runeCount int,
	matchIndex func(in string) []int,
	validate func(value string, previousValues []string) (error, []Info, []Datum)) Input {
	return Input{runeCount: runeCount, matchIndex: matchIndex, validate: validate}
}

func (i *Input) validateValue() {
	i.err, i.infos, i.data = i.validate(i.value, i.previousValues)
}

func (i *Input) isValidFmt() bool {
	if i.runeCount == 0 {
		return false
	}
	return utf8.RuneCountInString(i.value) == i.runeCount
}

// Info is structured information with formatted text.
type Info struct {
	Title string
	Text  string
}
