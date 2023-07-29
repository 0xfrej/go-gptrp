package main

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"gptrp/internal/config"
	"gptrp/internal/gpt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

func main() {
	cfg := loadConfig()
	if cfg == nil {
		log.Fatal("could not load config")
		return
	}

	args := getArgsWithoutVars()

	if len(args) == 0 {
		log.Fatal("Not enough arguments. Usage: <command> <args>")
	}

	if args[0] == "talk" {
		var scenario *config.Scenario
		if len(args) < 2 {
			scenario = showListOfScenarios(cfg)
		}

		if scenario == nil {
			scenarioArg := args[1]
			s, err := cfg.GetScenario(scenarioArg)
			if err != nil {
				log.Fatal("Scenario does not exist")
				return
			}
			scenario = &s
		}

		g, err := gpt.NewGpt(cfg, gpt.Settings{
			Scenario:         scenario.Name,
			Model:            openai.GPT3Dot5Turbo,
			MaxTokens:        120,
			StoreGptMessages: true,
		})
		if err != nil {
			log.Fatal(err)
		}
		runTalkLoop(g, cfg)
	}
}

func loadConfig() *config.Config {
	cfg := &config.Config{}
	if err := cfg.GetConfig(getConfigPath()); err != nil {
		log.Fatal(err)
		return nil
	}
	return cfg
}

func getConfigPath() string {
	configPath, ok := findVar("config")
	if ok {
		return strings.Split(configPath, "=")[1]
	} else {
		basePath, err := os.UserConfigDir()
		if err == nil {
			panic("could not get user config dir")
		}
		return path.Join(basePath, ".gptrp", "config.yaml")
	}
}

func getArgsWithoutVars() []string {
	var vars []string
	for _, v := range os.Args[1:] {
		if !strings.Contains(v, "=") {
			vars = append(vars, v)
		}
	}
	return vars
}

func findVar(name string) (string, bool) {
	prefix := name + "="
	for _, v := range os.Args[1:] {
		if strings.HasPrefix(v, prefix) {
			return v, true
		}
	}
	return "", false
}

func showListOfScenarios(cfg *config.Config) *config.Scenario {
	for {
		println("Available scenarios:")
		for i, s := range cfg.Scenarios {
			println("\n" + strconv.Itoa(i+1) + ")")
			println("\tName: " + s.Name)
			println("\tDescription: " + s.Description)
		}

		print("Choose scenario: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			log.Print(err)
		}
		atoi, err := strconv.Atoi(input)
		if err != nil {
			return nil
		}
		if atoi < 1 || atoi > len(cfg.Scenarios) {
			continue
		}

		return &cfg.Scenarios[atoi-1]
	}
}
