package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Formatter writes test results as Allure report.
type Formatter struct {
	mu          sync.Mutex
	Container   *Container
	Res         *Result
	LastTime    TimestampMs
	ResultsPath string
}

// WriteResult writes single result.
func (f *Formatter) WriteResult(r *Result) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.LastTime = GetTimestampMs()

	r.Stage = "finished"
	if r.Stop == 0 {
		r.Stop = f.LastTime
	}

	f.Container.Children = append(f.Container.Children, r.UUID)

	f.WriteJSON(fmt.Sprintf("%s-result.json", r.UUID), r)
}

// WriteJSON writes named value as JSON result.
func (f *Formatter) WriteJSON(name string, v interface{}) {
	j, err := json.Marshal(v)
	if err != nil {
		log.Fatal("failed to marshal json value:", err)
	}

	if err := ioutil.WriteFile(f.ResultsPath+"/"+name, j, 0o600); err != nil {
		log.Fatal("failed to write a file:", err)
	}
}

// Init prepares results directory.
func (f *Formatter) Init() error {
	err := os.MkdirAll(f.ResultsPath, 0o700)
	if err != nil {
		return fmt.Errorf("failed create allure results directory: %w", err)
	}

	return nil
}

// Finish flushes collected results.
func (f *Formatter) Finish(exec Executor) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.Res != nil {
		f.WriteResult(f.Res)
	}

	if f.Container.Stop == 0 {
		f.Container.Stop = GetTimestampMs()
	}

	f.WriteJSON(f.Container.UUID+"-container.json", f.Container)

	// Populate from env vars.
	if exec.Name == "" {
		exec.Name = os.Getenv("ALLURE_EXECUTOR_NAME")
		exec.Type = os.Getenv("ALLURE_EXECUTOR_TYPE")
		exec.URL = os.Getenv("ALLURE_EXECUTOR_URL")
		exec.BuildOrder, _ = strconv.Atoi(os.Getenv("ALLURE_EXECUTOR_BUILD_ORDER")) //nolint:errcheck
		exec.BuildName = os.Getenv("ALLURE_EXECUTOR_BUILD_NAME")
		exec.BuildURL = os.Getenv("ALLURE_EXECUTOR_BUILD_URL")
		exec.ReportName = os.Getenv("ALLURE_EXECUTOR_REPORT_NAME")
		exec.ReportURL = os.Getenv("ALLURE_EXECUTOR_REPORT_URL")
	}

	if exec.Name != "" {
		f.WriteJSON("executor.json", exec)
	}

	var env []byte

	for _, l := range os.Environ() {
		if strings.HasPrefix(l, "ALLURE_ENV_") {
			env = append(env, []byte(strings.TrimPrefix(l, "ALLURE_ENV_")+"\n")...)
		}
	}

	if len(env) > 0 {
		if err := ioutil.WriteFile(f.ResultsPath+"/environment.properties", env, 0o600); err != nil {
			log.Fatal("failed to write a file:", err)
		}
	}
}

// StartNewResult finishes previous Result and starts new.
func (f *Formatter) StartNewResult(res Result) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.Res != nil {
		f.WriteResult(f.Res)
	}

	f.LastTime = GetTimestampMs()

	if res.UUID == "" {
		res.UUID = uuid.New().String()
	}

	if res.Start == 0 {
		res.Start = f.LastTime
	}

	f.Res = &res
}

// StepFinished updates result with a finished step.
func StepFinished(res *Result, name string, status Status, statusDetails *StatusDetails, prepareStep func(s *Step), start TimestampMs) Step {
	step := Step{
		Name:          name,
		Status:        status,
		Stage:         "finished",
		Start:         start,
		StatusDetails: statusDetails,
	}

	step.Stop = GetTimestampMs()

	if status != Skipped || res.Status == "" {
		res.Status = status
	}

	if statusDetails != nil {
		res.StatusDetails = statusDetails
	}

	if prepareStep != nil {
		prepareStep(&step)
	}

	res.Steps = append(res.Steps, step)

	return step
}

// StepFinished finishes step and updates result.
func (f *Formatter) StepFinished(name string, status Status, statusDetails *StatusDetails, prepareStep func(s *Step)) {
	f.mu.Lock()
	defer f.mu.Unlock()

	step := StepFinished(f.Res, name, status, statusDetails, prepareStep, f.LastTime)

	f.LastTime = step.Stop
}

// BytesAttachment creates scalar attachment.
func (f *Formatter) BytesAttachment(content []byte, mediaType string) (*Attachment, error) {
	if mediaType == "" {
		mediaType = "text/plain"
	}

	att, err := NewAttachment("Doc", mediaType, f.ResultsPath, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	return att, nil
}

// TableAttachment creates table attachment.
func (f *Formatter) TableAttachment(table [][]string) (*Attachment, error) {
	mt := CSVMime
	buf := bytes.NewBuffer(nil)
	c := csv.NewWriter(buf)

	for _, row := range table {
		if err := c.Write(row); err != nil {
			log.Fatal("failed write csv row:", err)
		}
	}

	c.Flush()

	att, err := NewAttachment("Table", mt, f.ResultsPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed create table attachment: %w", err)
	}

	return att, nil
}
