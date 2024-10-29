package allure

import (
	"io"
	"log"
	"os"
	"strconv"
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

	mu          sync.Mutex
	threads     int
	busyThreads map[int]bool
	scenarios   map[*godog.Scenario]*scenarioContext
}

// TestRunStarted prepares test result directory.
func (f *formatter) TestRunStarted() {
	if err := f.Init(); err != nil {
		log.Fatal(err)
	}
}

type scenarioContext struct {
	result        *report.Result
	start         report.TimestampMs
	lastTime      report.TimestampMs
	totalSteps    int
	finishedSteps int
	thread        int
}

// Pickle receives scenario.
func (f *formatter) Pickle(scenario *godog.Scenario) {
	feature := f.Storage.MustGetFeature(scenario.Uri)
	res := report.Result{
		Name:        scenario.Name,
		HistoryID:   feature.Feature.Name + ": " + scenario.Name + scenario.Id,
		FullName:    scenario.Uri + ":" + scenario.Name,
		Description: scenario.Uri,
		Labels: []report.Label{
			{Name: "feature", Value: feature.Feature.Name},
			{Name: "suite", Value: f.Container.Name},
			{Name: "framework", Value: "godog"},
			{Name: "language", Value: "Go"},
		},
		Start: report.GetTimestampMs(),
		UUID:  uuid.New().String(),
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.scenarios == nil {
		f.scenarios = make(map[*godog.Scenario]*scenarioContext)
	}

	now := report.GetTimestampMs()
	sc := &scenarioContext{
		result:     &res,
		start:      now,
		lastTime:   now,
		totalSteps: len(scenario.Steps),
	}

	for thread, busy := range f.busyThreads {
		if !busy {
			f.busyThreads[thread] = true
			sc.thread = thread

			break
		}
	}

	if sc.thread == 0 {
		f.threads++
		sc.thread = f.threads

		if f.busyThreads == nil {
			f.busyThreads = make(map[int]bool)
		}

		f.busyThreads[f.threads] = true
	}

	f.scenarios[scenario] = sc

	sc.result.Labels = append(sc.result.Labels, report.Label{Name: "thread", Value: "routine " + strconv.Itoa(sc.thread)})
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

func (f *formatter) step(sc *godog.Scenario, st *godog.Step, status report.Status, statusDetails *report.StatusDetails) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.scenarios[sc]
	c.finishedSteps++

	step := report.StepFinished(c.result, st.Text, status, statusDetails, func(s *report.Step) {
		if att := f.argumentAttachment(st); att != nil {
			s.Attachments = append(s.Attachments, *att)
		}
	}, c.lastTime)

	f.LastTime = step.Stop

	if c.finishedSteps == c.totalSteps {
		f.WriteResult(c.result)
		f.busyThreads[c.thread] = false
	}
}

func statusDetails(prefix string, sd *godog.StepDefinition) *report.StatusDetails {
	if sd == nil || sd.Expr == nil {
		return nil
	}

	return &report.StatusDetails{
		Message: prefix + sd.Expr.String(),
	}
}

// Passed captures passed step.
func (f *formatter) Passed(sc *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	f.step(sc, st, report.Passed, nil)
}

// Skipped captures skipped step.
func (f *formatter) Skipped(sc *godog.Scenario, st *godog.Step, sd *godog.StepDefinition) {
	f.step(sc, st, report.Skipped, statusDetails("Skipped: ", sd))
}

// Undefined captures undefined step.
func (f *formatter) Undefined(sc *godog.Scenario, st *godog.Step, sd *godog.StepDefinition) {
	f.step(sc, st, report.Broken, statusDetails("Undefined: ", sd))
}

// Failed captures failed step.
func (f *formatter) Failed(sc *godog.Scenario, st *godog.Step, _ *godog.StepDefinition, err error) {
	f.step(sc, st, report.Failed, &report.StatusDetails{
		Message: err.Error(),
	})
}

// Pending captures pending step.
func (f *formatter) Pending(sc *godog.Scenario, st *godog.Step, sd *godog.StepDefinition) {
	f.step(sc, st, report.Unknown, statusDetails("Pending: ", sd))
}

// Summary finishes report.
func (f *formatter) Summary() {
	f.Finish(Exec)
}
