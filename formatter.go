package allure

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/formatters"
	"github.com/godogx/allure/report"
	"github.com/google/uuid"
)

var (
	// Exec allows configuring execution context information.
	Exec report.Executor

	// ResultsPath controls report destination.
	ResultsPath = os.Getenv("ALLURE_RESULTS_PATH")

	formatterRegister sync.Once
)

// RegisterFormatter adds allure to available formatters.
func RegisterFormatter() {
	formatterRegister.Do(func() {
		if ResultsPath == "" {
			ResultsPath = "./allure-results"
		}

		godog.Format("allure", "Allure formatter.",
			func(suite string, writer io.Writer) formatters.Formatter {
				if suite == "" {
					suite = "Features"
				}

				return &formatter{
					Formatter: report.Formatter{
						ResultsPath: strings.TrimSuffix(ResultsPath, "/"),
						Container: &report.Container{
							UUID:  uuid.New().String(),
							Start: report.GetTimestampMs(),
							Name:  suite,
						},
					},
					BaseFmt: godog.NewBaseFmt(suite, writer),
				}
			})
	})
}

type formatter struct {
	report.Formatter

	*godog.BaseFmt
}

// TestRunStarted prepares test result directory.
func (f *formatter) TestRunStarted() {
	if err := f.Init(); err != nil {
		log.Fatal(err)
	}
}

// Pickle receives scenario.
func (f *formatter) Pickle(scenario *godog.Scenario) {
	feature := f.Storage.MustGetFeature(scenario.Uri)
	res := report.Result{
		Name:        scenario.Name,
		HistoryID:   feature.Feature.Name + ": " + scenario.Name,
		FullName:    scenario.Uri + ":" + scenario.Name,
		Description: scenario.Uri,
		Labels: []report.Label{
			{Name: "feature", Value: feature.Feature.Name},
			{Name: "suite", Value: f.Container.Name},
			{Name: "framework", Value: "godog"},
			{Name: "language", Value: "Go"},
		},
	}

	f.StartNewResult(res)
}

func (f *formatter) argumentAttachment(st *godog.Step) *report.Attachment {
	if st.Argument == nil {
		return nil
	}

	if st.Argument.DocString != nil {
		att, err := f.BytesAttachment([]byte(st.Argument.DocString.Content), report.MediaType(st.Argument.DocString.MediaType))
		if err != nil {
			log.Fatal(err)
		}

		return att
	} else if st.Argument.DataTable != nil {
		var table [][]string

		for _, r := range st.Argument.DataTable.Rows {
			var rec []string
			for _, cell := range r.Cells {
				rec = append(rec, cell.Value)
			}
			table = append(table, rec)
		}

		att, err := f.TableAttachment(table)
		if err != nil {
			log.Fatal(err)
		}

		return att
	}

	return nil
}

func (f *formatter) step(st *godog.Step, status report.Status, statusDetails *report.StatusDetails) {
	f.StepFinished(st.Text, status, statusDetails, func(s *report.Step) {
		if att := f.argumentAttachment(st); att != nil {
			s.Attachments = append(s.Attachments, *att)
		}
	})
}

// Passed captures passed step.
func (f *formatter) Passed(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	f.step(st, report.Passed, nil)
}

// Skipped captures skipped step.
func (f *formatter) Skipped(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	f.step(st, report.Skipped, nil)
}

// Undefined captures undefined step.
func (f *formatter) Undefined(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	f.step(st, report.Broken, nil)
}

// Failed captures failed step.
func (f *formatter) Failed(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition, err error) {
	f.step(st, report.Failed, &report.StatusDetails{
		Message: err.Error(),
	})
}

// Pending captures pending step.
func (f *formatter) Pending(*godog.Scenario, *godog.Step, *godog.StepDefinition) {
}

// Summary finishes report.
func (f *formatter) Summary() {
	f.Finish(Exec)
}
