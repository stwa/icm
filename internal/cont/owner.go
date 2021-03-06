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

// Owner has a code and associated company with its location in the form of country and city.
type Owner struct {
	Code    string
	Company string
	City    string
	Country string
}

// IsOwnerCode checks if string is three upper case letters.
func IsOwnerCode(code string) error {
	if len(code) != 3 {
		return NewErrContValidate(fmt.Sprintf("%s is not 3 letters long", code))
	}
	if !isUpperLetter(code) {
		return NewErrContValidate(fmt.Sprintf("%s is not 3 upper case letters", code))
	}
	return nil
}
