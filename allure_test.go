package allure_test

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/godogx/allure"
	"github.com/stretchr/testify/assert"
)

func TestRegisterFormatter(t *testing.T) {
	allure.RegisterFormatter()

	out := bytes.NewBuffer(nil)

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			s.Step("I pass", func() {})
			s.Step("I sleep for a bit", func() {
				time.Sleep(time.Duration(rand.Float64() * float64(time.Second))) //nolint:gosec
			})
			s.Step("I fail", func() error { return errors.New("failed") })
		},
		Options: &godog.Options{
			Format:      "allure",
			Output:      out,
			NoColors:    true,
			Paths:       []string{"_testdata"},
			Concurrency: 10,
		},
	}

	st := suite.Run()
	assert.Equal(t, 1, st) // Failed.
}
