//
// Copyright 2021 Shawn Black
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"bytes"
	"fmt"
	"github.com/common-nighthawk/go-figure"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"
)

type Configuration struct {
	CommitMessage struct {
		Commit struct {
			Enabled           bool   `yaml:"enabled"`
			VerificationRegEx string `yaml:"verificationRegEx"`
		} `yaml:"commit"`
	} `yaml:"commitMessage"`
	PrepareCommitMessage struct {
		Commit struct {
			Enabled  bool   `yaml:"enabled"`
			Template string `yaml:"template"`
		} `yaml:"commit"`
	} `yaml:"prepareCommitMessage"`
	PrePush struct {
		Enabled                                     bool     `yaml:"enabled"`
		EnforceProtectedBranchesOnNonExistentRemote bool     `yaml:"enforceProtectedBranchesOnNonExistentRemote"`
		ProtectedBranches                           []string `yaml:"protectedBranches"`
		ValidBranches                               []string `yaml:"validBranches"`
	} `yaml:"prePush"`
	PreCommit struct {
		Enabled bool               `yaml:"enabled"`
		Execute []PreCommitExecute `yaml:"execute"`
	} `yaml:"preCommit"`
}

type PreCommitExecute struct {
	Command   string   `yaml:"command"`
	Arguments []string `yaml:"arguments"`
}

func main() {
	const configFile = ".git-smart.yaml"
	const codeName = "agent-86"
	fileInfo, configErr := os.Stat(configFile)
	if os.IsNotExist(configErr) {
		fmt.Printf("%s does not exist; gracefully exiting\n", configFile)
		return
	}
	configData, configErr := ioutil.ReadFile(fileInfo.Name())
	if configErr != nil {
		panic(configErr)
	}
	var configuration Configuration
	configErr = yaml.Unmarshal(configData, &configuration)
	if configErr != nil {
		panic(configErr)
	}
	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "git-smart",
		Usage:                "manage git expectations",
		Compiled:             time.Now(),
		Authors: []*cli.Author{
			{
				Name:  "Shawn Black",
				Email: "shawn@castleblack.us",
			},
		},
		Copyright: "(c) 2021 Castle Black",
		Commands: []*cli.Command{
			{
				Name:  "setup",
				Usage: "setup git hooks",
				Action: func(c *cli.Context) error {
					if c.Bool("header") {
						renderHeader(codeName)
					}
					fmt.Println("This is where we would setup the git hooks")
					return nil
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "header",
						Usage: "display the header",
					},
				},
			},
			{
				Name:  "pre-commit",
				Usage: "git pre-commit hook",
				Action: func(c *cli.Context) error {
					if !configuration.PreCommit.Enabled {
						return nil
					}
					for _, execute := range configuration.PreCommit.Execute {
						cmd := exec.Command(execute.Command, execute.Arguments...)
						var stdBuffer bytes.Buffer
						multiWriter := io.MultiWriter(os.Stdout, &stdBuffer)
						cmd.Stdout = multiWriter
						cmd.Stderr = multiWriter
						if preCommitError := cmd.Run(); preCommitError != nil {
							panic(preCommitError)
						}
					}
					return nil
				},
			},
			{
				Name:  "pre-push",
				Usage: "git pre-push hook",
				Action: func(c *cli.Context) error {
					remote := ""
					if c.Args().Len() > 0 {
						remote = c.Args().First()
					}
					if !configuration.PrePush.Enabled {
						return nil
					}
					pwd, prePushError := os.Getwd()
					if prePushError != nil {
						panic(prePushError)
					}
					repo, prePushError := git.PlainOpen(pwd)
					if prePushError != nil {
						panic(prePushError)
					}
					headRef, prePushError := repo.Head()
					if prePushError != nil {
						panic(prePushError)
					}
					if !headRef.Name().IsBranch() {
						return nil
					}

					references, prePushError := repo.References()
					if prePushError != nil {
						panic(prePushError)
					}

					remoteBranchExists := false

					prePushError = references.ForEach(func(ref *plumbing.Reference) error {
						if ref.Type() == plumbing.SymbolicReference {
							return nil
						}
						if ref.Name().IsRemote() {
							remoteBranchExists = ref.Name().Short() == fmt.Sprintf("%s/%s", remote, headRef.Name().Short())
						}
						return nil
					})

					checkProtectedBranch := remoteBranchExists || configuration.PrePush.EnforceProtectedBranchesOnNonExistentRemote

					if checkProtectedBranch {
						for _, protectedBranchRegex := range configuration.PrePush.ProtectedBranches {
							re := regexp.MustCompile(protectedBranchRegex)
							result := re.MatchString(headRef.Name().Short())
							if result {
								ec := cli.Exit("Current branch is protected", 1)
								return ec
							}
						}
					}

					validBranch := false
					for _, validBranchRegex := range configuration.PrePush.ValidBranches {
						re := regexp.MustCompile(validBranchRegex)
						result := re.MatchString(headRef.Name().Short())
						if result {
							validBranch = true
							break
						}
					}
					if !validBranch {
						ec := cli.Exit("Current branch does not meet naming requirements", 1)
						return ec
					}
					return nil
				},
			},
			{
				Name:  "prepare-commit-msg",
				Usage: "git prepare-commit-msg hook",
				Action: func(c *cli.Context) error {
					commitTemplate := ""
					commitType := "commit"
					switch c.Args().Len() {
					case 1:
						commitTemplate = c.Args().Get(0)
					case 2:
						commitTemplate = c.Args().Get(0)
						commitType = c.Args().Get(1)
					case 3:
						commitTemplate = c.Args().Get(0)
						commitType = c.Args().Get(1)
					default:
						ec := cli.Exit("No commit template referenced", 1)
						return ec
					}
					commitTypeValid := false
					template := ""
					switch commitType {
					case "commit":
						if !configuration.PrepareCommitMessage.Commit.Enabled {
							return nil
						}
						template = configuration.PrepareCommitMessage.Commit.Template
						commitTypeValid = true
					}
					if !commitTypeValid {
						return nil
					}
					templateFile, templateError := os.Create(commitTemplate)
					if templateError != nil {
						ec := cli.Exit(templateError.Error(), 1)
						return ec
					}
					_, templateError = templateFile.WriteString(template)
					if templateError != nil {
						ec := cli.Exit(templateError.Error(), 1)
						return ec
					}
					templateError = templateFile.Close()
					fmt.Printf("Template written to %s\n", commitTemplate)
					return nil
				},
			},
			{
				Name:  "commit-msg",
				Usage: "git commit-msg hook",
				Action: func(c *cli.Context) error {
					commitMessage := ""
					commitType := "commit"
					switch c.Args().Len() {
					case 1:
						commitMessage = c.Args().Get(0)
					case 2:
						commitMessage = c.Args().Get(0)
						commitType = c.Args().Get(1)
					case 3:
						commitMessage = c.Args().Get(0)
						commitType = c.Args().Get(1)
					default:
						ec := cli.Exit("No commit message referenced", 1)
						return ec
					}
					commitTypeValid := false
					verificationRegEx := ""
					switch commitType {
					case "commit":
						if !configuration.CommitMessage.Commit.Enabled {
							return nil
						}
						verificationRegEx = configuration.CommitMessage.Commit.VerificationRegEx
						commitTypeValid = true
					}
					if !commitTypeValid {
						return nil
					}
					commitData, commitError := ioutil.ReadFile(commitMessage)
					if commitError != nil {
						ec := cli.Exit(commitError.Error(), 1)
						return ec
					}
					re := regexp.MustCompile(verificationRegEx)
					result := re.MatchString(string(commitData))
					if !result {
						return cli.Exit("Commit message does not match required regular expression", 100)
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func renderHeader(codeName string) {
	header := figure.NewColorFigure("git smart", "", "blue", true)
	header.Print()
	header = figure.NewColorFigure(codeName, "doom", "green", true)
	header.Print()
	fmt.Println()
}
