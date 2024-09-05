package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/deckhouse/rbacgen/internal/engine"
	"github.com/spf13/cobra"
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
