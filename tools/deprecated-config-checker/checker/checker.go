package checker

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

type RawYaml map[string]interface{}

type Checker struct {
	input      RawYaml
	deprecates RawYaml
	deletes    RawYaml
}

func loadYAMLFile(path string) (RawYaml, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var out RawYaml
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

func (c *Checker) CheckDeprecated() []DeprecationNotes {
	fieldsInInput := enumerateCursedFields(c.deprecates, c.input, "", []DeprecationNotes{})
	return fieldsInInput
}

func (c *Checker) CheckDeleted() []DeprecationNotes {
	fieldsInInput := enumerateCursedFields(c.deletes, c.input, "", []DeprecationNotes{})
	return fieldsInInput
}

type deprecationAnnotation struct {
	deprecatedValues []string
	msg              string
}

func getDeprecationAnnotation(value interface{}) (deprecationAnnotation, bool) {
	// If the value is a string, return it as the message
	if msg, is := value.(string); is {
		return deprecationAnnotation{
			msg: msg,
		}, true
	}

	// If the value is a map, check if it has a _msg field
	if inner, is := value.(RawYaml); is {
		msg, exists := inner[messageField]
		if !exists {
			return deprecationAnnotation{}, false
		}

		var deprecatedValues []string
		if v, exists := inner[deprecatedValuesField]; exists {
			asIfcSlice := v.([]interface{})
			deprecatedValues = make([]string, len(asIfcSlice))
			for i, v := range asIfcSlice {
				deprecatedValues[i] = v.(string)
			}
		}

		return deprecationAnnotation{
			msg:              msg.(string),
			deprecatedValues: deprecatedValues,
		}, true
	}

	return deprecationAnnotation{}, false
}

type DeprecationNotes struct {
	deprecationAnnotation
	itemPath  string
	itemValue string
}

func (d DeprecationNotes) String() string {
	var sb strings.Builder

	sb.WriteString(d.itemPath)
	if d.itemValue != "" {
		sb.WriteString(" = ")
		sb.WriteString(d.itemValue)
	}
	sb.WriteString(": " + d.msg)
	if len(d.deprecatedValues) > 0 {
		sb.WriteString("\n\t|- " + "Deprecated values: ")
		sb.WriteString(strings.Join(d.deprecatedValues, ", "))
	}

	return sb.String()
}

func appenToPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func enumerateCursedFields(deprecates, input RawYaml, rootPath string, deprecations []DeprecationNotes) []DeprecationNotes {
	for key, deprecate := range deprecates {
		inputValue, exists := input[key]
		if !exists {
			// If this item is not set in the input, we can skip it.
			continue
		}

		path := appenToPath(rootPath, key)

		note, isDeprecatedNote := getDeprecationAnnotation(deprecate)
		if isDeprecatedNote {
			var inputDeprecated bool

			// If there are no specific values deprecated, the whole config is deprecated.
			// Otherwise, look for the input value in the list of deprecated values.
			if len(note.deprecatedValues) == 0 {
				inputDeprecated = true
			} else {
				for _, v := range note.deprecatedValues {
					if v == inputValue {
						inputDeprecated = true
						break
					}
				}
			}

			if inputDeprecated {
				var itemValueStr string
				if v, is := inputValue.(string); is {
					itemValueStr = v
				}

				deprecations = append(deprecations, DeprecationNotes{
					deprecationAnnotation: note,
					itemPath:              path,
					itemValue:             itemValueStr,
				})
				continue
			}
		}

		// To this point, the deprecate item is not a leaf, so we need to recurse into it.
		if deprecateYaml, is := deprecate.(RawYaml); is {
			switch v := inputValue.(type) {
			case RawYaml:
				deprecations = enumerateCursedFields(deprecateYaml, v, path, deprecations)
			case []interface{}:
				// If the input is a list, recurse into each item.
				for i, item := range v {
					itemYaml := item.(RawYaml)
					deprecations = enumerateCursedFields(deprecateYaml, itemYaml, appenToPath(path, fmt.Sprintf("[%d]", i)), deprecations)
				}
			}
		}
	}

	return deprecations
}
