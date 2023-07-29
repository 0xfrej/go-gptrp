package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

const ScenarioNotFoundError = "scenario not found"

type Scenario struct {
	Name                string `yaml:"name"`
	Description         string `yaml:"description"`
	NarratorPersonality string `yaml:"narrator_personality"`
	WorldBuilding       string `yaml:"world_building"`
	DungeonRoomBuilding string `yaml:"dungeon_room_building"`
}

type Config struct {
	Clipboard struct {
		WordWrap bool `yaml:"word_wrap"`
		MaxSize  int  `yaml:"max_size"`
	}
	OpenAI struct {
		ApiKey string `yaml:"api_key"`
	} `yaml:"openai"`
	Scenarios []Scenario
}

func (c *Config) GetScenario(scenario string) (Scenario, error) {
	for _, s := range c.Scenarios {
		if s.Name == scenario {
			return s, nil
		}
	}
	return Scenario{}, errors.New(ScenarioNotFoundError)
}

func (c *Config) GetConfig(path string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ToString() string {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
