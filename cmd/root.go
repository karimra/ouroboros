/*
Copyright © 2021 Karim Radhouani <medkarimrdi@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/karimra/ouroboros/orbrs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var oApp = orbrs.New()

func newRootCmd() *cobra.Command {
	oApp.RootCmd = &cobra.Command{
		Use:     "ouroboros",
		Aliases: []string{"orbrs"},
		Short:   "ouroboros is a closed loop automation tool",

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if oApp.Config.Flags.LogFile != "" {
				f, err := os.OpenFile(oApp.Config.Flags.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					return err
				}
				oApp.SetLogOutput(f)

			} else if oApp.Config.Flags.Debug || oApp.Config.FileConfig.GetBool("debug") {
				oApp.SetLogLevel(log.DebugLevel)
				oApp.SetLogOutput(os.Stderr)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			oApp.Start()
		},
	}
	oApp.InitFlags()
	return oApp.RootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(newRootCmd().Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	err := oApp.Config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed loading config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "using config file: %q\n", oApp.Config.FileConfig.ConfigFileUsed())
}
