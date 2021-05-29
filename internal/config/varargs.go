package config

import "fmt"

type varArgs []string

func (i *varArgs) String() string {
	return fmt.Sprintf("%v", []string(*i))
}

func (i *varArgs) Set(value string) error {
	*i = append(*i, value)
	return nil
}
