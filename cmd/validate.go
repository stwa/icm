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

package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/meyermarcel/icm/configs"
	"github.com/meyermarcel/icm/internal/cont"
	"github.com/meyermarcel/icm/internal/data"
	"github.com/meyermarcel/icm/internal/input"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const patternModesInfo string = `                    auto = matches automatically a pattern
        container-number = matches a container number
                   owner = matches a three letter owner code
owner-equipment-category = matches a three letter owner code with equipment category ID
               size-type = matches length, width+height and type code`

const (
	auto                   = "auto"
	containerNumber        = "container-number"
	owner                  = "owner"
	ownerEquipmentCategory = "owner-equipment-category"
	sizeType               = "size-type"
)

var (
	yellow    = color.New(color.FgYellow).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()
	bold      = color.New(color.Bold).SprintFunc()
	underline = color.New(color.Underline).SprintFunc()
)

type patternValue struct {
	pflag.Flag
	value    string
	patterns map[string]newPattern
}

func (p *patternValue) String() string {
	return p.value
}

func (p *patternValue) Set(value string) error {
	if pattern := p.patterns[value]; pattern == nil {
		return fmt.Errorf("%s is not \n%s", value, patternModesInfo)
	}
	p.value = value
	return nil
}

func (*patternValue) Type() string {
	return "mode"
}

func newPatternValue() *patternValue {
	return &patternValue{
		value: auto,
		patterns: map[string]newPattern{
			auto:                   newAutoPattern,
			containerNumber:        newContNumPattern,
			owner:                  newOwnerPattern,
			ownerEquipmentCategory: newOwnerEquipCatPattern,
			sizeType:               newSizeTypePattern,
		},
	}
}

type newPattern func(decoders decoders) [][]input.Input

func (p *patternValue) newPattern(value string) newPattern {
	return p.patterns[value]
}

var pValue = newPatternValue()

func newValidateCmd(writer, writerErr io.Writer, viperCfg *viper.Viper, decoders decoders) *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate intermodal container markings",
		Long: `Validate intermodal container markings.

` + sepHelp,
		Example: `  ` + appName + ` validate 'ABCU 1234560'

  ` + appName + ` validate 'ABCU'

  ` + appName + ` validate '20G1'

  ` + appName + ` validate --` + configs.SepOE + ` '' --` + configs.SepSC + ` '' 'ABCU 1234560'
  
  ` + appName + ` validate --pattern container-number 'ABCU 123456'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			reader := strings.NewReader(args[0])
			bufReader := bufio.NewReader(reader)
			peek, err := bufReader.Peek(bufReader.Buffered())
			if err != nil {
				return err
			}
			patternStr := viperCfg.GetString(configs.Pattern)
			pattern := pValue.newPattern(patternStr)(decoders)
			validator := input.NewValidator(pattern)

			if isSingleLine(string(peek)) {
				buf := new(bytes.Buffer)
				_, _ = buf.ReadFrom(reader)
				inputs := validator.Validate(buf.String())
				fancyPrinter := input.NewFancyPrinter(writer, inputs)
				fancyPrinter.SetIndent("  ")

				switch patternStr {
				case auto, containerNumber, owner, ownerEquipmentCategory:
					if len(inputs) == 3 {
						fancyPrinter.SetSeparators(
							"",
							viperCfg.GetString(configs.SepST),
						)
					} else {
						fancyPrinter.SetSeparators(
							viperCfg.GetString(configs.SepOE),
							viperCfg.GetString(configs.SepES),
							viperCfg.GetString(configs.SepSC),
							viperCfg.GetString(configs.SepCS),
							"",
							viperCfg.GetString(configs.SepST),
						)
					}

				case sizeType:
					fancyPrinter.SetSeparators(
						"",
						viperCfg.GetString(configs.SepST),
					)
				}

				err := fancyPrinter.Print()
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	validateCmd.Flags().VarP(pValue, configs.Pattern, "p",
		fmt.Sprintf("sets pattern matching mode to\n%s\n", patternModesInfo))
	validateCmd.Flags().String(configs.SepOE, "",
		"ABC(*)U1234560   20G1  (*) separates owner code and equipment category id")
	validateCmd.Flags().String(configs.SepES, "",
		"ABCU(*)1234560   20G1  (*) separates equipment category id and serial number")
	validateCmd.Flags().String(configs.SepSC, "",
		"ABCU123456(*)0   20G1  (*) separates serial number and check digit")
	validateCmd.Flags().String(configs.SepCS, "",
		"ABCU1234560 (*)  20G1  (*) separates check digit and size")
	validateCmd.Flags().String(configs.SepST, "",
		"ABCU1234560   20(*)G1  (*) separates size and type")
	err := viperCfg.BindPFlags(validateCmd.Flags())
	writeErr(writerErr, err)

	return validateCmd
}

func isSingleLine(s string) bool {
	scanner := bufio.NewScanner(strings.NewReader(s))
	counter := 0
	for scanner.Scan() {
		counter++
		if counter > 1 {
			return false
		}
	}
	return true
}

func newAutoPattern(decoders decoders) [][]input.Input {
	owner := newOwnerInput(decoders.ownerDecodeUpdater)
	equipCat := newEquipCatInput(decoders.equipCatDecoder)
	serialNum := newSerialNumInput()
	checkDigit := newCheckDigitInput()
	length := newLengthInput(decoders.lengthDecoder)
	heightWidth := newHeightWidthInput(decoders.heightWidthDecoder)
	typeAndGroup := newTypeAndGroupInput(decoders.typeDecoder)

	return [][]input.Input{
		{owner, equipCat, serialNum, checkDigit, length, heightWidth, typeAndGroup},
		{owner, equipCat, serialNum, checkDigit},
		{owner, equipCat},
		{owner},
		{length, heightWidth, typeAndGroup},
	}
}

func newContNumPattern(decoders decoders) [][]input.Input {
	owner := newOwnerInput(decoders.ownerDecodeUpdater)
	equipCat := newEquipCatInput(decoders.equipCatDecoder)
	serialNum := newSerialNumInput()
	checkDigit := newCheckDigitInput()

	return [][]input.Input{
		{owner, equipCat, serialNum, checkDigit},
	}
}

func newOwnerPattern(decoders decoders) [][]input.Input {
	owner := newOwnerInput(decoders.ownerDecodeUpdater)

	return [][]input.Input{
		{owner},
	}
}

func newOwnerEquipCatPattern(decoders decoders) [][]input.Input {
	owner := newOwnerInput(decoders.ownerDecodeUpdater)
	equipCat := newEquipCatInput(decoders.equipCatDecoder)

	return [][]input.Input{
		{owner, equipCat},
	}
}

func newSizeTypePattern(decoders decoders) [][]input.Input {
	length := newLengthInput(decoders.lengthDecoder)
	heightWidth := newHeightWidthInput(decoders.heightWidthDecoder)
	typeAndGroup := newTypeAndGroupInput(decoders.typeDecoder)

	return [][]input.Input{
		{length, heightWidth, typeAndGroup},
	}
}

func newOwnerInput(ownerDecodeUpdater data.OwnerDecodeUpdater) input.Input {
	owner := input.NewInput(
		3,
		regexp.MustCompile(`[A-Za-z]{3}`).FindStringIndex,
		func(value string, previousValues []string) (bool, []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s long (e.g. %s)",
					underline("owner code"),
					bold("3 letters"),
					underline(ownerDecodeUpdater.GenerateRandomCodes(1)[0]))}}
			}
			found, owner := ownerDecodeUpdater.Decode(value)
			if !found {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s (e.g. %s)",
					underline(value),
					bold("registered"),
					underline(ownerDecodeUpdater.GenerateRandomCodes(1)[0]))}}

			}
			return true, []input.Info{{Text: owner.Company},
				{Text: owner.City},
				{Text: owner.Country}}
		})
	owner.SetToUpper()
	return owner
}

func newEquipCatInput(equipCatDecoder data.EquipCatDecoder) input.Input {
	equipCat := input.NewInput(
		1,
		regexp.MustCompile(`[A-Za-z]`).FindStringIndex,
		func(value string, previousValues []string) (bool, []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s",
					underline("equipment category id"),
					equipCatIDsAsList(equipCatDecoder))}}
			}

			found, cat := equipCatDecoder.Decode(value)
			if !found {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s",
					underline("equipment category id"),
					equipCatIDsAsList(equipCatDecoder))}}
			}
			return true, []input.Info{{Text: cat.Info}}
		})
	equipCat.SetToUpper()
	return equipCat
}

func equipCatIDsAsList(equipCatDecoder data.EquipCatDecoder) string {

	b := strings.Builder{}

	iDs := equipCatDecoder.AllCatIDs()
	sort.Strings(iDs)
	for i, element := range iDs {
		b.WriteString(green(element))
		if i < len(iDs)-2 {
			b.WriteString(", ")
		}
		if i == len(iDs)-2 {
			b.WriteString(" or ")
		}
	}
	return b.String()
}

func newSerialNumInput() input.Input {
	return input.NewInput(
		6,
		regexp.MustCompile(`\d{6}`).FindStringIndex,
		func(value string, previousValues []string) (bool, []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s long",
					underline("serial number"),
					bold("6 numbers"))}}
			}
			return true, nil
		})
}

func newCheckDigitInput() input.Input {
	return input.NewInput(
		1,
		regexp.MustCompile(`\d`).FindStringIndex,
		func(value string, previousValues []string) (bool, []input.Info) {
			if len(strings.Join(previousValues[0:3], "")) != 10 {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not calculable",
					underline("check digit"))}}
			}

			checkDigit := cont.CalcCheckDigit(previousValues[2], previousValues[1], previousValues[0])

			number, err := strconv.Atoi(value)
			if err != nil {
				return false, appendCheckDigit10Info(checkDigit,
					[]input.Info{{Text: fmt.Sprintf("%s must be a %s (calculated: %s)",
						underline("check digit"),
						bold("number"),
						green(checkDigit))}})
			}

			if number != checkDigit%10 {
				return false, appendCheckDigit10Info(checkDigit, []input.Info{{Text: fmt.Sprintf(
					"calculated %s is %s",
					underline("check digit"),
					green(checkDigit%10))}})
			}

			return true, appendCheckDigit10Info(checkDigit, nil)
		})
}

func appendCheckDigit10Info(checkDigit int, infos []input.Info) []input.Info {
	if checkDigit == 10 {
		if infos == nil {
			infos = make([]input.Info, 0)
		}
		infos = append(infos, input.Info{
			Text: fmt.Sprintf("It is not recommended to use a %s", underline("serial number"))})
		infos = append(infos, input.Info{
			Text: fmt.Sprintf("that generates %s %s (0).", underline("check digit"), yellow(10))})
	}
	return infos
}

func newLengthInput(lengthDecoder data.LengthDecoder) input.Input {
	length := input.NewInput(
		1,
		regexp.MustCompile(`[A-Za-z\d]`).FindStringIndex,
		func(value string, previousValues []string) (bool, []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not a %s or a %s",
					underline("length code"),
					bold("valid number"),
					bold("valid character"))}}
			}

			found, length := lengthDecoder.Decode(value)
			if !found {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s",
					underline("length code"),
					bold("valid"))}}
			}
			return true, []input.Info{{Text: fmt.Sprintf("length: %s", length.Length)}}
		})
	length.SetToUpper()
	return length
}

func newHeightWidthInput(heightWidthDecoder data.HeightWidthDecoder) input.Input {
	heightWidth := input.NewInput(
		1,
		regexp.MustCompile(`[A-Za-z\d]`).FindStringIndex,
		func(value string, previousValues []string) (b bool, infos []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not a %s or a %s",
					underline("height and width code"),
					bold("valid number"),
					bold("valid character"))}}
			}

			found, heightWidth := heightWidthDecoder.Decode(value)
			if !found {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s",
					underline("height and width code"),
					bold("valid"))}}
			}
			return true, []input.Info{
				{Text: fmt.Sprintf("height: %s", heightWidth.Height)},
				{Text: fmt.Sprintf("width:  %s", heightWidth.Width)}}
		})
	heightWidth.SetToUpper()
	return heightWidth
}

func newTypeAndGroupInput(typeDecoder data.TypeDecoder) input.Input {
	typeAndGroup := input.NewInput(
		2,
		regexp.MustCompile(`[A-Za-z\d]{2}`).FindStringIndex,
		func(value string, previousValues []string) (b bool, infos []input.Info) {
			if value == "" {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not a %s or a %s",
					underline("type code"),
					bold("valid number"),
					bold("valid character"))}}
			}

			found, typeAndGroup := typeDecoder.Decode(value)
			if !found {
				return false, []input.Info{{Text: fmt.Sprintf("%s is not %s",
					underline("type code"),
					bold("valid"))}}
			}
			return true, []input.Info{
				{Text: fmt.Sprintf("type:  %s", typeAndGroup.TypeInfo)},
				{Text: fmt.Sprintf("group: %s", typeAndGroup.GroupInfo)}}

		})
	typeAndGroup.SetToUpper()
	return typeAndGroup
}
