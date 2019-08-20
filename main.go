package main

import (
	"context"
	"encoding/json"
	"os"
	"regexp"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

var (
	authtoken     string
	validSchedule = regexp.MustCompile("^[A-Z0-9]{4,10}$").MatchString
)

const (
	authTokenEnvironmentName = "PAGERDUTY_API_TOKEN"
)

type onCallUserResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Response is just shorthand for response object
type Response events.APIGatewayProxyResponse

func init() {
	var ok bool
	authtoken, ok = os.LookupEnv(authTokenEnvironmentName)
	if !ok {
		logrus.Fatal(authTokenEnvironmentName + " environment variable is not set; required to talk to PagerDuty API")
	}
}

// HandlerWithLogging is our entrypoint, invoked for each API call
func HandlerWithLogging(ctx context.Context, req events.APIGatewayProxyRequest) (Response, error) {

	baseLogger := logrus.New()
	baseLogger.SetFormatter(&logrus.JSONFormatter{})
	baseLogger.SetLevel(logrus.DebugLevel)

	requestLogger := baseLogger.WithField("requestID", "Req"+req.RequestContext.RequestID)

	return handler(ctx, requestLogger, req)
}

// handler builds responses to "who's on call" requests
func handler(ctx context.Context, logger *logrus.Entry, req events.APIGatewayProxyRequest) (Response, error) {
	client := pagerduty.NewClient(authtoken)

	var schedule = req.QueryStringParameters["schedule"]
	logger = logger.WithField("schedule", schedule)

	if !validSchedule(schedule) {
		logger.Error("Received invalid schedule")
		return errorResponse("Unable to determine on-call person")
	}

	logger.Info("Calling pagerduty API")
	onCall, err := client.ListOnCalls(pagerduty.ListOnCallOptions{Includes: []string{"users"}, ScheduleIDs: []string{schedule}})
	if err != nil {
		logger.Error("Unable to request on-call person: ", err)
		return errorResponse("Unable to determine on-call person")
	}

	if len(onCall.OnCalls) < 1 {
		logger.Error("No users returned in request")
		return errorResponse("No on-call person")
	}

	user := onCall.OnCalls[0].User

	whosOnCall := onCallUserResponse{Name: user.Name, Email: user.Email}

	resp, err := json.Marshal(whosOnCall)
	if err != nil {
		logger.Error("Unable to serialize response object", err)
		return errorResponse("Unable to build response")
	}

	return Response{
		Body:       string(resp),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, nil
}

func errorResponse(message string) (Response, error) {
	errorResponse := struct {
		Message string `json:"error-message"`
	}{message}
	resp, err := json.Marshal(errorResponse)
	if err != nil {
		panic(err)
	}

	return Response{
		Body:       string(resp),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(HandlerWithLogging)
}
