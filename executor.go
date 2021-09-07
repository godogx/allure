package allure

// Executor describes execution context.
type Executor struct {
	Name string `json:"name,omitempty" example:"Jenkins"`
	// Type may be one of [github, gitlab, teamcity, bamboo, jenkins] or a custom one.
	Type       string `json:"type,omitempty" example:"jenkins"`
	URL        string `json:"url,omitempty" example:"url"`
	BuildOrder int    `json:"buildOrder,omitempty" example:"13"`
	BuildName  string `json:"buildName,omitempty" example:"allure-report_deploy#13"`
	BuildURL   string `json:"buildUrl,omitempty" example:"http://example.org/build#13"`
	ReportURL  string `json:"reportUrl,omitempty" example:"http://example.org/build#13/AllureReport"`
	ReportName string `json:"reportName,omitempty" example:"Demo allure report"`
}

//{
//  "name": "Jenkins",
//  "type": "jenkins",
//  "url": "http://example.org",
//  "buildOrder": 13,
//  "buildName": "allure-report_deploy#13",
//  "buildUrl": "http://example.org/build#13",
//  "reportUrl": "http://example.org/build#13/AllureReport",
//  "reportName": "Demo allure report"
//}
