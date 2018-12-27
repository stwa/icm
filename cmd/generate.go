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
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/meyermarcel/icm/configs"
	"github.com/meyermarcel/icm/internal/cont"
	"github.com/meyermarcel/icm/internal/data"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ownerValue struct {
	value string
}

func (o *ownerValue) String() string {
	return o.value
}

func (o *ownerValue) Set(value string) error {
	if err := cont.IsOwnerCode(value); err != nil {
		return err
	}
	o.value = value
	return nil
}

func (*ownerValue) Type() string {
	return "owner code"
}

type serialNumValue struct {
	value int
}

func (r *serialNumValue) String() string {
	return strconv.Itoa(r.value)
}

func (r *serialNumValue) Set(value string) error {
	serialNum, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if serialNum < 0 || serialNum > 999999 {
		return fmt.Errorf("%d is not in range from 0 to 999999", serialNum)
	}
	r.value = serialNum
	return nil
}

func (*serialNumValue) Type() string {
	return "int"
}

func newGenerateCmd(writer, writerErr io.Writer, viper *viper.Viper, ownerDecoder data.OwnerDecoder) *cobra.Command {

	var count int
	var startValue = serialNumValue{}
	var endValue = serialNumValue{}
	var ownerValue = ownerValue{}
	var excludeCheckDigit10 bool

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a random container number",
		Long: `Owners specified in

  ` + filepath.Join("$HOME", appDir) + `

are used. Container numbers with check digit 10 are excluded.
Equipment category ID 'U' is used for every container number.

` + sepHelp,
		Example: `  ` + appName + ` generate

  ` + appName + ` generate --count 5000

  ` + appName + ` generate --` + configs.SepOE + ` '' --` + configs.SepSC + ` ''`,
		Args: cobra.NoArgs,
		// https://github.com/spf13/viper/issues/233
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlag(configs.SepOE, cmd.Flags().Lookup(configs.SepOE)); err != nil {
				return err
			}
			if err := viper.BindPFlag(configs.SepES, cmd.Flags().Lookup(configs.SepES)); err != nil {
				return err
			}
			return viper.BindPFlag(configs.SepSC, cmd.Flags().Lookup(configs.SepSC))
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			builder := cont.NewUniqueGeneratorBuilder().
				Count(count).
				ExcludeCheckDigit10(excludeCheckDigit10)

			if cmd.Flags().Changed("owner") {
				builder.OwnerCodes([]string{ownerValue.value})
			} else {
				builder.OwnerCodes(ownerDecoder.GetAllOwnerCodes())
			}

			if cmd.Flags().Changed("start") {
				builder.Start(startValue.value)
			}

			if cmd.Flags().Changed("end") {
				builder.End(endValue.value)
			}

			generator, err := builder.Build()
			if err != nil {
				return err
			}
			for generator.Generate() {
				contNumber := generator.ContNum()
				_, err := io.WriteString(writer,
					fmt.Sprintf("%s%s%s%s%s%s%d\n",
						contNumber.OwnerCode(), viper.GetString(configs.SepOE),
						contNumber.EquipCatID(), viper.GetString(configs.SepES),
						contNumber.SerialNumber(), viper.GetString(configs.SepSC),
						contNumber.CheckDigit()))
				writeErr(writerErr, err)
			}
			return nil
		},
	}

	generateCmd.Flags().SortFlags = false

	generateCmd.Flags().IntVarP(&count, "count", "c", 1, "count of container numbers")
	generateCmd.Flags().VarP(&startValue, "start", "s", "start of serial number range")
	generateCmd.Flags().VarP(&endValue, "end", "e", "end of serial number range")
	generateCmd.Flags().Var(&ownerValue, "owner", "custom owner code")
	generateCmd.Flags().BoolVar(&excludeCheckDigit10, "exclude-check-digit-10", false, "exclude check digit 10")

	generateCmd.Flags().String(configs.SepOE, configs.SepOEDefVal,
		"ABC(*)U1234560  (*) separates owner code and equipment category id")
	generateCmd.Flags().String(configs.SepES, configs.SepESDefVal,
		"ABCU(*)1234560  (*) separates equipment category id and serial number")
	generateCmd.Flags().String(configs.SepSC, configs.SepSCDefVal,
		"ABCU123456(*)0  (*) separates serial number and check digit")

	return generateCmd
}
