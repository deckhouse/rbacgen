// Copyright 2024 Flant JSC
//
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

package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/deckhouse/rbacgen/internal/engine"
)

func init() {
	root.AddCommand(generateCmd)
}

var root = &cobra.Command{
	Use:   "rbacgen",
	Short: "rbacgen - a tool to generate RBACv2 roles",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use some command")
	},
}

func Execute() {
	if err := root.Execute(); err != nil {
		log.Printf("failed to execute: '%s'", err)
		os.Exit(1)
	}
}

var generateCmd = &cobra.Command{
	Use:     "generate",
	Short:   "Generate roles and docs by walking over the specific dir",
	Example: "rbacgen generate ee docs.yaml - to generate roles only from ee dir\nrbacgen generate . docs.yaml - to generate roles from the current dir",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 2 {
			return errors.New("workdir and docs path are required")
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments")
		}
		return engine.WalkAndRender(context.Background(), args[0], args[1])
	},
}
