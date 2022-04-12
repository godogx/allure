package allure_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"

	"github.com/godogx/allure"
)

func TestRegisterFormatter(t *testing.T) {
	allure.RegisterFormatter()

	out := bytes.NewBuffer(nil)

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			s.Step("I pass", func() {})
			s.Step("I fail", func() error { return errors.New("failed") })
		},
		Options: &godog.Options{
			Format:   "allure",
			Output:   out,
			NoColors: true,
			Paths:    []string{"_testdata"},
		},
	}

	st := suite.Run()
	assert.Equal(t, 1, st) // Failed.
}
