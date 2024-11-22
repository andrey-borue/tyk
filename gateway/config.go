package gateway

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

var cfg = Config{}

func init() {
	if err := env.Parse(&cfg); err != nil {
		fmt.Println("Error loading analytics config")
		fmt.Print(err)
		return
	}
	fmt.Println("Analytics config loaded")
}

type Config struct {
	ResponseCodeFilterEnable bool  `env:"ANALYTIC_RESPONSE_CODE_FILTER_ENABLE" envDefault:"true"`
	ResponseCodeFilterList   []int `env:"ANALYTIC_RESPONSE_CODE_FILTER_LIST" envDefault:"500,501,502,503,504,404,403,400,413,409,401,405"`
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
