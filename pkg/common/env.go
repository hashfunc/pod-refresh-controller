package common

import (
	"os"

	"github.com/samber/mo"
)

func GetEnv(key string) mo.Option[string] {
	return mo.TupleToOption(os.LookupEnv(key))
}
