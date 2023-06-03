package devnetutils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	libcommon "github.com/ledgerwatch/erigon-lib/common"

	"github.com/ledgerwatch/erigon/crypto"
	"github.com/ledgerwatch/log/v3"
)

// ClearDevDB cleans up the dev folder used for the operations
func ClearDevDB(dataDir string, logger log.Logger) error {
	logger.Info("Deleting nodes' data folders")

	nodeNumber := 1
	for {
		nodeDataDir := filepath.Join(dataDir, fmt.Sprintf("%d", nodeNumber))
		fileInfo, err := os.Stat(nodeDataDir)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}
		if fileInfo.IsDir() {
			if err := os.RemoveAll(nodeDataDir); err != nil {
				return err
			}
			logger.Info("SUCCESS => Deleted", "datadir", nodeDataDir)
		} else {
			break
		}
		nodeNumber++
	}
	return nil
}

// UniqueIDFromEnode returns the unique ID from a node's enode, removing the `?discport=0` part
func UniqueIDFromEnode(enode string) (string, error) {
	if len(enode) == 0 {
		return "", fmt.Errorf("invalid enode string")
	}

	// iterate through characters in the string until we reach '?'
	// using index iteration because enode characters have single codepoints
	var i int
	for i < len(enode) && enode[i] != byte('?') {
		i++
	}

	// if '?' is not found in the enode, return the original enode
	if i == len(enode) {
		return enode, nil
	}

	return enode[:i], nil
}

// NamespaceAndSubMethodFromMethod splits a parent method into namespace and the actual method
func NamespaceAndSubMethodFromMethod(method string) (string, string, error) {
	parts := strings.SplitN(method, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid string to split")
	}
	return parts[0], parts[1], nil
}

func GenerateTopic(signature string) []libcommon.Hash {
	hashed := crypto.Keccak256([]byte(signature))
	return []libcommon.Hash{libcommon.BytesToHash(hashed)}
}

// RandomNumberInRange returns a random number between min and max NOT inclusive
func RandomNumberInRange(min, max uint64) (uint64, error) {
	if max <= min {
		return 0, fmt.Errorf("Invalid range: upper bound %d less or equal than lower bound %d", max, min)
	}

	diff := int64(max - min)

	n, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return 0, err
	}

	return uint64(n.Int64() + int64(min)), nil
}

type Args []string

func AsArgs(args interface{}) (Args, error) {

	argsValue := reflect.ValueOf(args)

	if argsValue.Kind() == reflect.Ptr {
		argsValue = argsValue.Elem()
	}

	if argsValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Args type must be struct or struc pointer, got %T", args)
	}

	return gatherArgs(argsValue, func(v reflect.Value, field reflect.StructField) (string, error) {
		tag := field.Tag.Get("arg")

		if tag == "-" {
			return "", nil
		}

		// only process public fields (reflection won't return values of unsafe fields without unsafe operations)
		if r, _ := utf8.DecodeRuneInString(field.Name); !(unicode.IsLetter(r) && unicode.IsUpper(r)) {
			return "", nil
		}

		var key string
		var positional bool

		for _, key = range strings.Split(tag, ",") {
			if key == "" {
				continue
			}

			key = strings.TrimLeft(key, " ")

			if pos := strings.Index(key, ":"); pos != -1 {
				key = key[:pos]
			}

			switch {
			case strings.HasPrefix(key, "---"):
				return "", fmt.Errorf("%s.%s: too many hyphens", v.Type().Name(), field.Name)
			case strings.HasPrefix(key, "--"):

			case strings.HasPrefix(key, "-"):
				if len(key) != 2 {
					return "", fmt.Errorf("%s.%s: short arguments must be one character only", v.Type().Name(), field.Name)
				}
			case key == "positional":
				key = ""
				positional = true
			default:
				return "", fmt.Errorf("unrecognized tag '%s' on field %s", key, tag)
			}
		}

		if len(key) == 0 && !positional {
			key = "--" + strings.ToLower(field.Name)
		}

		value := fmt.Sprintf("%v", v.FieldByIndex(field.Index).Interface())

		flagValue, isFlag := field.Tag.Lookup("flag")

		if isFlag {
			if value != "true" {
				if flagValue == "true" {
					value = flagValue
				}
			}
		}

		if len(value) == 0 {
			if defaultString, hasDefault := field.Tag.Lookup("default"); hasDefault {
				value = defaultString
			}

			if len(value) == 0 {
				return "", nil
			}
		}

		if len(key) == 0 {
			return value, nil
		}

		if isFlag {
			if value == "true" {
				return key, nil
			}

			return "", nil
		}

		if len(value) == 0 {
			return key, nil
		}

		return fmt.Sprintf("%s=%s", key, value), nil
	})
}

func gatherArgs(v reflect.Value, visit func(v reflect.Value, field reflect.StructField) (string, error)) (args Args, err error) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)

		var gathered Args

		fieldType := field.Type

		if fieldType.Kind() == reflect.Ptr {
			fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			gathered, err = gatherArgs(v.FieldByIndex(field.Index), visit)
		} else {
			var value string

			if value, err = visit(v, field); len(value) > 0 {
				gathered = Args{value}
			}
		}

		if err != nil {
			return nil, err
		}

		args = append(args, gathered...)
	}

	return args, nil
}
