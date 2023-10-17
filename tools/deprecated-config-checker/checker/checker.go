package checker

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	deprecatedValuesField = "_deprecated"
	messageField          = "_msg"
)

type CheckerConfig struct {
	DeprecatesFile string
	DeletesFile    string
	ConfigFile     string
}

func (c *CheckerConfig) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&c.DeprecatesFile, "deprecates-file", "tools/deprecated-config-checker/deprecated-config.yaml", "YAML file with deprecated configs")
	f.StringVar(&c.DeletesFile, "deletes-file", "tools/deprecated-config-checker/deleted-config.yaml", "YAML file with deleted configs")
	f.StringVar(&c.ConfigFile, "config.file", "", "User-defined config file to validate")
}

func (c *CheckerConfig) Validate() error {
	if c.ConfigFile == "" {
		return fmt.Errorf("config.file is required")
	}
	return nil
}

type Config map[string]interface{}

type Checker struct {
	input      Config
	deprecates Config
	deletes    Config
}

func loadYAMLFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var out Config
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func NewChecker(cfg CheckerConfig) (*Checker, error) {
	input, err := loadYAMLFile(cfg.ConfigFile)
	if err != nil {
		return nil, err
	}

	deprecates, err := loadYAMLFile(cfg.DeprecatesFile)
	if err != nil {
		return nil, err
	}

	deletes, err := loadYAMLFile(cfg.DeletesFile)
	if err != nil {
		return nil, err
	}

	return &Checker{
		input:      input,
		deprecates: deprecates,
		deletes:    deletes,
	}, nil
}

func (c *Checker) CheckDeprecated() []string {
	fieldsInInput := enumerateCursedFields(c.deprecates, c.input, "", []string{})
	return fieldsInInput
}

func (c *Checker) CheckDeleted() []string {
	fieldsInInput := enumerateCursedFields(c.deletes, c.input, "", []string{})
	return fieldsInInput
}

type DeprecationNotes struct {
	deprecatedValues []string
	msg              string
}

func GetDeprecationNotes(value interface{}) (DeprecationNotes, bool) {
	// If the value is a string, return it as the message
	if msg, is := value.(string); is {
		return DeprecationNotes{
			msg: msg,
		}, true
	}

	// If the value is a map, check if it has a _msg field
	if inner, is := value.(Config); is {
		msg, exists := inner[messageField]
		if !exists {
			return DeprecationNotes{}, false
		}

		var deprecatedValues []string
		if v, exists := inner[deprecatedValuesField]; exists {
			asIfcSlice := v.([]interface{})
			deprecatedValues = make([]string, len(asIfcSlice))
			for i, v := range asIfcSlice {
				deprecatedValues[i] = v.(string)
			}
		}

		return DeprecationNotes{
			msg:              msg.(string),
			deprecatedValues: deprecatedValues,
		}, true
	}

	return DeprecationNotes{}, false
}

func enumerateCursedFields(deprecates, input Config, prefix string, fieldsInA []string) []string {
	for key := range deprecates {
		fullKey := prefix + key

		inInput, exists := input[key]
		if !exists {
			continue
		}

		note, isDeprecatedNote := GetDeprecationNotes(deprecates[key])
		if isDeprecatedNote {
			// If notes.deprecatedValues is not empty, look for the input value in the list of deprecated values.

			fmt.Printf("Found deprecated config %s: %v\n Got: %v", fullKey, note, inInput)
		}

		// If cursed is a map and input is an array of maps, iterate though each item looking for the cursed.

		if subConfCursed, is := deprecates[key].(Config); is {
			if subConfInput, is := inInput.(Config); is {
				fieldsInA = enumerateCursedFields(subConfCursed, subConfInput, fullKey+".", fieldsInA)
			}
		}
	}

	// for key := range a {
	// 	fullKey := prefix + key
	// 	if _, exists := b[key]; exists {
	// 		fieldsInA = append(fieldsInA, fullKey)
	// 	}
	//
	// 	if aSubConfig, aIsConfig := a[key].(Config); aIsConfig {
	// 		if bSubConfig, bIsConfig := b[key].(Config); bIsConfig {
	// 			fieldsInA = enumerateFields(aSubConfig, bSubConfig, fullKey+".", fieldsInA)
	// 		}
	// 	}
	// }

	return fieldsInA
}
