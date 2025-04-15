package venom

import (
    "fmt"
	"os"
	"time"
	"net/url"
	"bytes"
	_ "embed"
	"encoding/json"
	"text/template"
	"github.com/pkg/errors"
)

//go:embed venom_output.html
var templateHTML string

type TestsHTML struct {
	Tests     Tests  `json:"tests"`
	JSONValue string `json:"jsonValue"`
}

func buildOpenSearchURL(baseURL, fromDate, toDate, namespace string) string {
    query := ""

    if namespace != "" {
        query = fmt.Sprintf("source:%s*", namespace)
    }

    encodedQuery := url.QueryEscape(query)

    return fmt.Sprintf("%s/app/data-explorer/discover#?_a=(discover:(columns:!(application_name,client_ip,domain_id,full_msg),isDirty:!f,sort:!()),metadata:(indexPattern:'%s',view:discover))&_g=(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:'%s',to:'%s'))&_q=(filters:!(),query:(language:kuery,query:'%s'))",
        baseURL,
        os.Getenv("OPENSEARCH_LOGS_STREAM_ID"),
        fromDate,
        toDate,
        encodedQuery)
}

func outputHTML(testsResult *Tests) ([]byte, error) {
	var buf bytes.Buffer

    hasLogsPlatform := os.Getenv("HAS_LOGS_PLATFORM") == "true"
    logsPlatformBaseURL := os.Getenv("LOGS_PLATFORM_BASE_URL")
    namespace := os.Getenv("NAMESPACE")
    if hasLogsPlatform && logsPlatformBaseURL != "" {
        for _, suite := range testsResult.TestSuites {
            for _, testCase := range suite.TestCases {
                for idx := range testCase.TestStepResults {
                    result := &testCase.TestStepResults[idx]
                    if len(result.Errors) > 0 && !result.End.IsZero() {

                        fromDate := result.End.Add(-1 * time.Minute).UTC().Format("2006-01-02T15:04:05.000Z")
                        toDate := result.End.UTC().Format("2006-01-02T15:04:05.000Z")

                        result.LogsUrl = buildOpenSearchURL(logsPlatformBaseURL, fromDate, toDate, namespace)
                    }
                }
            }
        }
    }

	testJSON, err := json.MarshalIndent(testsResult, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "unable to make json value")
	}

	testsHTML := TestsHTML{
		Tests:     *testsResult,
		JSONValue: string(testJSON),
	}
	tmpl := template.Must(template.New("reportHTML").Parse(templateHTML))
	if err := tmpl.Execute(&buf, testsHTML); err != nil {
		return nil, errors.Wrap(err, "unable to make template")
	}
	return buf.Bytes(), nil
}
