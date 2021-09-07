package allure

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/formatters"
	"github.com/google/uuid"
)

var (
	// Exec allows configuring execution context information.
	Exec Executor

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
					resultsPath: strings.TrimSuffix(ResultsPath, "/"),
					container: &Container{
						UUID:  uuid.New().String(),
						Start: getTimestampMs(),
						Name:  suite,
					},
					BaseFmt: godog.NewBaseFmt(suite, writer),
				}
			})
	})
}

type formatter struct {
	container   *Container
	res         *Result
	lastTime    TimestampMs
	resultsPath string

	*godog.BaseFmt
}

func (f *formatter) writeResult(r *Result) {
	f.lastTime = getTimestampMs()

	r.Stage = "finished"
	r.Stop = f.lastTime
	f.container.Children = append(f.container.Children, r.UUID)

	f.writeJSON(fmt.Sprintf("%s-result.json", r.UUID), r)
}

// TestRunStarted prepares test result directory.
func (f *formatter) TestRunStarted() {
	err := os.MkdirAll(f.resultsPath, 0o700)
	if err != nil {
		log.Fatal("failed create allure results directory:", err)
	}
}

// Pickle receives scenario.
func (f *formatter) Pickle(scenario *godog.Scenario) {
	if f.res != nil {
		f.writeResult(f.res)
	}

	f.lastTime = getTimestampMs()

	feature := f.Storage.MustGetFeature(scenario.Uri)

	f.res = &Result{
		UUID:        uuid.New().String(),
		Name:        scenario.Name,
		HistoryID:   feature.Feature.Name + ": " + scenario.Name,
		FullName:    scenario.Uri + ":" + scenario.Name,
		Description: scenario.Uri,
		Start:       f.lastTime,
		Labels: []Label{
			{Name: "feature", Value: feature.Feature.Name},
			{Name: "suite", Value: f.container.Name},
			{Name: "framework", Value: "godog"},
			{Name: "language", Value: "Go"},
		},
	}
}

func getTimestampMs() TimestampMs {
	return TimestampMs(time.Now().UnixNano() / int64(time.Millisecond))
}

const (
	csvMime = "text/csv"
)

func mediaType(t string) string {
	switch t {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "csv":
		return csvMime
	default:
		return "text/plain"
	}
}

func (f *formatter) argumentAttachment(st *godog.Step) *Attachment {
	if st.Argument == nil {
		return nil
	}

	if st.Argument.DocString != nil {
		att, err := NewAttachment("Doc", mediaType(st.Argument.DocString.MediaType),
			f.resultsPath, []byte(st.Argument.DocString.Content))
		if err != nil {
			log.Fatal("failed to create attachment:", err)
		}

		return att
	} else if st.Argument.DataTable != nil {
		mt := csvMime
		buf := bytes.NewBuffer(nil)
		c := csv.NewWriter(buf)

		for _, r := range st.Argument.DataTable.Rows {
			var rec []string
			for _, cell := range r.Cells {
				rec = append(rec, cell.Value)
			}
			if err := c.Write(rec); err != nil {
				log.Fatal("failed write csv row:", err)
			}
		}
		c.Flush()

		att, err := NewAttachment("Table", mt, f.resultsPath, buf.Bytes())
		if err != nil {
			log.Fatal("failed create table attachment:", err)
		}

		return att
	}

	return nil
}

func (f *formatter) step(st *godog.Step) Step {
	step := Step{
		Name:  st.Text,
		Stage: "finished",
		Start: f.lastTime,
	}

	if att := f.argumentAttachment(st); att != nil {
		step.Attachments = append(step.Attachments, *att)
	}

	f.lastTime = getTimestampMs()
	step.Stop = f.lastTime

	return step
}

// Passed captures passed step.
func (f *formatter) Passed(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	step := f.step(st)
	step.Status = Passed
	f.res.Steps = append(f.res.Steps, step)
	f.res.Status = Passed
}

// Skipped captures skipped step.
func (f *formatter) Skipped(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	step := f.step(st)
	step.Status = Skipped
	f.res.Steps = append(f.res.Steps, step)
}

// Undefined captures undefined step.
func (f *formatter) Undefined(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition) {
	step := f.step(st)
	step.Status = Broken

	f.res.Steps = append(f.res.Steps, step)
}

// Failed captures failed step.
func (f *formatter) Failed(_ *godog.Scenario, st *godog.Step, _ *godog.StepDefinition, err error) {
	details := &StatusDetails{
		Message: err.Error(),
	}

	step := f.step(st)
	step.Status = Failed
	step.StatusDetails = details

	f.res.Steps = append(f.res.Steps, step)
	f.res.Status = Failed
	f.res.StatusDetails = details
}

// Pending captures pending step.
func (f *formatter) Pending(*godog.Scenario, *godog.Step, *godog.StepDefinition) {
}

func (f *formatter) writeJSON(name string, v interface{}) {
	j, err := json.Marshal(v)
	if err != nil {
		log.Fatal("failed to marshal json value:", err)
	}

	if err := ioutil.WriteFile(f.resultsPath+"/"+name, j, 0o600); err != nil {
		log.Fatal("failed to write a file:", err)
	}
}

// Summary finishes report.
func (f *formatter) Summary() {
	if f.res != nil {
		f.writeResult(f.res)
	}

	f.container.Stop = getTimestampMs()

	f.writeJSON(f.container.UUID+"-container.json", f.container)

	// Populate from env vars.
	if Exec.Name == "" {
		Exec.Name = os.Getenv("ALLURE_EXECUTOR_NAME")
		Exec.Type = os.Getenv("ALLURE_EXECUTOR_TYPE")
		Exec.URL = os.Getenv("ALLURE_EXECUTOR_URL")
		Exec.BuildOrder, _ = strconv.Atoi(os.Getenv("ALLURE_EXECUTOR_BUILD_ORDER")) // nolint:errcheck
		Exec.BuildName = os.Getenv("ALLURE_EXECUTOR_BUILD_NAME")
		Exec.BuildURL = os.Getenv("ALLURE_EXECUTOR_BUILD_URL")
		Exec.ReportName = os.Getenv("ALLURE_EXECUTOR_REPORT_NAME")
		Exec.ReportURL = os.Getenv("ALLURE_EXECUTOR_REPORT_URL")
	}

	if Exec.Name != "" {
		f.writeJSON("executor.json", Exec)
	}

	var env []byte

	for _, l := range os.Environ() {
		if strings.HasPrefix(l, "ALLURE_ENV_") {
			env = append(env, []byte(strings.TrimPrefix(l, "ALLURE_ENV_")+"\n")...)
		}
	}

	if len(env) > 0 {
		if err := ioutil.WriteFile(f.resultsPath+"/environment.properties", env, 0o600); err != nil {
			log.Fatal("failed to write a file:", err)
		}
	}
}
