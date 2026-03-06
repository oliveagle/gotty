package utils

import (
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/fatih/structs"
	"github.com/yudai/hcl"

	"github.com/oliveagle/gotty/pkg/homedir"
)

func GenerateFlags(options ...interface{}) (flags []cli.Flag, mappings map[string]string, err error) {
	mappings = make(map[string]string)

	for _, struct_ := range options {
		o := structs.New(struct_)
		for _, field := range o.Fields() {
			flagName := field.Tag("flagName")
			if flagName == "" {
				continue
			}
			envName := "GOTTY_" + strings.ToUpper(strings.Join(strings.Split(flagName, "-"), "_"))
			mappings[flagName] = field.Name()

			flagShortName := field.Tag("flagSName")
			flagDescription := field.Tag("flagDescribe")

			switch field.Kind() {
			case reflect.String:
				sf := &cli.StringFlag{
					Name:    flagName,
					Usage:   flagDescription,
					EnvVars: []string{envName},
				}
				if flagShortName != "" {
					sf.Aliases = []string{flagShortName}
				}
				sf.Value = field.Value().(string)
				flags = append(flags, sf)
			case reflect.Bool:
				bf := &cli.BoolFlag{
					Name:    flagName,
					Usage:   flagDescription,
					EnvVars: []string{envName},
				}
				if flagShortName != "" {
					bf.Aliases = []string{flagShortName}
				}
				flags = append(flags, bf)
			case reflect.Int:
				ifval := field.Value().(int)
				iflag := &cli.IntFlag{
					Name:    flagName,
					Usage:   flagDescription,
					EnvVars: []string{envName},
					Value:   ifval,
				}
				if flagShortName != "" {
					iflag.Aliases = []string{flagShortName}
				}
				flags = append(flags, iflag)
			}
		}
	}

	return
}

func ApplyFlags(
	flags []cli.Flag,
	mappingHint map[string]string,
	c *cli.Context,
	options ...interface{},
) {
	objects := make([]*structs.Struct, len(options))
	for i, struct_ := range options {
		objects[i] = structs.New(struct_)
	}

	for flagName, fieldName := range mappingHint {
		if !c.IsSet(flagName) {
			continue
		}
		var field *structs.Field
		var ok bool
		for _, o := range objects {
			field, ok = o.FieldOk(fieldName)
			if ok {
				break
			}
		}
		if field == nil {
			continue
		}
		var val interface{}
		switch field.Kind() {
		case reflect.String:
			val = c.String(flagName)
		case reflect.Bool:
			val = c.Bool(flagName)
		case reflect.Int:
			val = c.Int(flagName)
		}
		field.Set(val)
	}
}

func ApplyConfigFile(filePath string, options ...interface{}) error {
	filePath = homedir.Expand(filePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}

	fileString := []byte{}
	log.Printf("Loading config file at: %s", filePath)
	fileString, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	for _, object := range options {
		if err := hcl.Decode(object, string(fileString)); err != nil {
			return err
		}
	}

	return nil
}
