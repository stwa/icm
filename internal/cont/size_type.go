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

package cont

import "fmt"

// HeightWidth describes width and height of first code in specified standard size code.
type HeightWidth struct {
	Width  string
	Height string
}

// Length describes length of second code in the specified standard size code.
type Length struct {
	Length string
}

// Type has code and information about the specified standard type.
type Type struct {
	TypeCode string
	TypeInfo string
}

// Group has code and information about an specified type group.
type Group struct {
	GroupCode string
	GroupInfo string
}

// TypeAndGroup encapsulates type and group.
type TypeAndGroup struct {
	Type
	Group
}

// IsLengthCode checks for correct format
func IsLengthCode(code string) error {
	return isOneUpperAlphanumericChar(code)
}

// IsHeightWidthCode checks if string is one upper case alphanumeric character.
func IsHeightWidthCode(code string) error {
	return isOneUpperAlphanumericChar(code)
}

// IsTypeCode checks if string is two upper case alphanumeric characters.
func IsTypeCode(code string) error {
	if len(code) != 2 {
		return NewErrContValidate(fmt.Sprintf("%s is not 2 characters long", code))
	}
	if !isUpperAlphanumeric(code) {
		return NewErrContValidate(
			fmt.Sprintf("%s is not 2 upper case alphanumeric characters", code))
	}
	return nil
}
